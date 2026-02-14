package vyosclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// newTestServer creates an httptest.Server that simulates VyOS API responses.
// The handler receives the parsed "data" form field and the endpoint path.
func newTestServer(t *testing.T, handler func(endpoint, data string) (int, apiResponse)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		data := r.FormValue("data")
		key := r.FormValue("key")

		if key != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success": false, "error": "Invalid API key"}`))
			return
		}

		statusCode, resp := handler(r.URL.Path, data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func successResponse(data any) apiResponse {
	raw, _ := json.Marshal(data)
	return apiResponse{Success: true, Data: raw}
}

func errorResponse(msg string) apiResponse {
	return apiResponse{Success: false, Error: &msg}
}

func TestNew(t *testing.T) {
	t.Parallel()
	c := New("https://vyos.example.com", "my-key")
	if c.baseURL != "https://vyos.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://vyos.example.com")
	}
	if c.apiKey != "my-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "my-key")
	}
}

func TestNew_WithHTTPClient(t *testing.T) {
	t.Parallel()
	custom := &http.Client{}
	c := New("https://vyos.example.com", "my-key", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Error("expected custom http client")
	}
}

func TestAuthError(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		return http.StatusOK, successResponse("ok")
	})
	defer srv.Close()

	c := New(srv.URL, "wrong-key")
	err := c.Set(t.Context(), []string{"system", "host-name"}, "test")
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T: %v", err, err)
	}
}

func TestMutexSerialization(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		n := concurrent.Add(1)
		for {
			old := maxConcurrent.Load()
			if n <= old || maxConcurrent.CompareAndSwap(old, n) {
				break
			}
		}
		// Simulate some work
		concurrent.Add(-1)
		return http.StatusOK, successResponse("ok")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Set(t.Context(), []string{"system", "host-name"}, "test")
		}()
	}
	wg.Wait()

	if observed := maxConcurrent.Load(); observed > 1 {
		t.Errorf("max concurrent requests = %d, want 1 (mutex not working)", observed)
	}
}

func TestHTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	err := c.Set(t.Context(), []string{"system", "host-name"}, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusBadGateway)
	}
}

func TestAPIErrorResponse(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		return http.StatusOK, errorResponse("Configuration path is not valid")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	err := c.Set(t.Context(), []string{"invalid", "path"}, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message != "Configuration path is not valid" {
		t.Errorf("message = %q, want %q", apiErr.Message, "Configuration path is not valid")
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		return http.StatusOK, successResponse("ok")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	err := c.Set(ctx, []string{"system", "host-name"}, "test")
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}
