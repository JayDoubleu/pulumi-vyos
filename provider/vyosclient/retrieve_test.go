package vyosclient

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestShowConfig(t *testing.T) {
	t.Parallel()

	configData := map[string]any{
		"host-name":   "router1",
		"domain-name": "example.com",
	}

	srv := newTestServer(t, func(endpoint, data string) (int, apiResponse) {
		if endpoint != "/retrieve" {
			t.Errorf("endpoint = %q, want /retrieve", endpoint)
		}
		var req map[string]any
		if err := json.Unmarshal([]byte(data), &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req["op"] != "showConfig" {
			t.Errorf("op = %v, want showConfig", req["op"])
		}
		return http.StatusOK, successResponse(configData)
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	result, err := c.ShowConfig(t.Context(), []string{"system"})
	if err != nil {
		t.Fatalf("ShowConfig() error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["host-name"] != "router1" {
		t.Errorf("host-name = %v, want router1", got["host-name"])
	}
}

func TestShowConfig_EmptyPath(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, func(_, data string) (int, apiResponse) {
		var req map[string]any
		if err := json.Unmarshal([]byte(data), &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		path, ok := req["path"].([]any)
		if !ok || len(path) != 0 {
			t.Errorf("path = %v, want empty array", req["path"])
		}
		return http.StatusOK, successResponse(map[string]any{"system": map[string]any{}})
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	_, err := c.ShowConfig(t.Context(), []string{})
	if err != nil {
		t.Fatalf("ShowConfig() error: %v", err)
	}
}

func TestExists_True(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, func(endpoint, data string) (int, apiResponse) {
		if endpoint != "/retrieve" {
			t.Errorf("endpoint = %q, want /retrieve", endpoint)
		}
		var req map[string]any
		if err := json.Unmarshal([]byte(data), &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req["op"] != "exists" {
			t.Errorf("op = %v, want exists", req["op"])
		}
		// VyOS returns {success: true, data: true} when path exists.
		return http.StatusOK, successResponse(true)
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	exists, err := c.Exists(t.Context(), []string{"system", "host-name"})
	if err != nil {
		t.Fatalf("Exists() error: %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}
}

func TestExists_False(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		// VyOS returns {success: true, data: false} when path doesn't exist.
		return http.StatusOK, successResponse(false)
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	exists, err := c.Exists(t.Context(), []string{"system", "nonexistent"})
	if err != nil {
		t.Fatalf("Exists() error: %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestExists_FalseErrorResponse(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		// Some VyOS versions return success=false for non-existent paths.
		return http.StatusOK, errorResponse("Configuration path does not exist")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	exists, err := c.Exists(t.Context(), []string{"system", "nonexistent"})
	if err != nil {
		t.Fatalf("Exists() error: %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestExists_UnknownAPIError_Propagates(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, func(_, _ string) (int, apiResponse) {
		// An unknown error with StatusCode==0 should not be swallowed.
		return http.StatusOK, errorResponse("internal processing error")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	_, err := c.Exists(t.Context(), []string{"system", "host-name"})
	if err == nil {
		t.Fatal("Exists() error = nil, want error for unknown API error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0", apiErr.StatusCode)
	}
}
