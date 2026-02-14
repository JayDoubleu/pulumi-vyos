package vyosclient

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
)

func TestSet(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var receivedData string
	srv := newTestServer(t, func(endpoint, data string) (int, apiResponse) {
		if endpoint != "/configure" {
			t.Errorf("endpoint = %q, want /configure", endpoint)
		}
		mu.Lock()
		receivedData = data
		mu.Unlock()
		return http.StatusOK, successResponse("ok")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	err := c.Set(t.Context(), []string{"system", "host-name"}, "router1")
	if err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	mu.Lock()
	data := receivedData
	mu.Unlock()

	var op Operation
	if err := json.Unmarshal([]byte(data), &op); err != nil {
		t.Fatalf("unmarshal sent data: %v", err)
	}
	if op.Op != "set" {
		t.Errorf("op = %q, want %q", op.Op, "set")
	}
	if op.Value != "router1" {
		t.Errorf("value = %v, want %q", op.Value, "router1")
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var receivedData string
	srv := newTestServer(t, func(endpoint, data string) (int, apiResponse) {
		if endpoint != "/configure" {
			t.Errorf("endpoint = %q, want /configure", endpoint)
		}
		mu.Lock()
		receivedData = data
		mu.Unlock()
		return http.StatusOK, successResponse("ok")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	err := c.Delete(t.Context(), []string{"interfaces", "dummy", "dum1"})
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	mu.Lock()
	data := receivedData
	mu.Unlock()

	var op Operation
	if err := json.Unmarshal([]byte(data), &op); err != nil {
		t.Fatalf("unmarshal sent data: %v", err)
	}
	if op.Op != "delete" {
		t.Errorf("op = %q, want %q", op.Op, "delete")
	}
}

func TestBatchConfigure(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var receivedData string
	srv := newTestServer(t, func(endpoint, data string) (int, apiResponse) {
		if endpoint != "/configure" {
			t.Errorf("endpoint = %q, want /configure", endpoint)
		}
		mu.Lock()
		receivedData = data
		mu.Unlock()
		return http.StatusOK, successResponse("ok")
	})
	defer srv.Close()

	c := New(srv.URL, "test-api-key")
	ops := []Operation{
		{Op: "set", Path: []any{"system", "host-name"}, Value: "router1"},
		{Op: "set", Path: []any{"system", "domain-name"}, Value: "example.com"},
		{Op: "delete", Path: []any{"interfaces", "dummy", "dum1"}},
	}
	err := c.BatchConfigure(t.Context(), ops)
	if err != nil {
		t.Fatalf("BatchConfigure() error: %v", err)
	}

	mu.Lock()
	data := receivedData
	mu.Unlock()

	var received []Operation
	if err := json.Unmarshal([]byte(data), &received); err != nil {
		t.Fatalf("unmarshal sent data: %v", err)
	}
	if len(received) != 3 {
		t.Errorf("len(ops) = %d, want 3", len(received))
	}
}
