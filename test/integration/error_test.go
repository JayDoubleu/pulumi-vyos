//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestError_AuthFailure(t *testing.T) {
	// Use a valid client first to confirm the VM is reachable.
	_ = vyosClient(t)

	client := vyosClientWithKey(t, "wrong-api-key")
	ctx := context.Background()

	_, err := client.ShowConfig(ctx, []string{"system"})
	if err == nil {
		t.Fatal("expected auth error with wrong API key, got nil")
	}

	var authErr *vyosclient.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *vyosclient.AuthError, got %T: %v", err, err)
	}
}

func TestError_ShowConfigNonExistent(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	// Querying a path that does not exist in VyOS config.
	_, err := client.ShowConfig(ctx, []string{"interfaces", "dummy", "nonexistent99"})
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}

	// VyOS returns an API error for non-existent config paths.
	var apiErr *vyosclient.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *vyosclient.APIError, got %T: %v", err, err)
	}
	t.Logf("ShowConfig non-existent: %v", apiErr)
}

func TestError_DeleteNonExistent(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	// VyOS silently succeeds when deleting a path that does not exist.
	err := client.Delete(ctx, []string{"interfaces", "dummy", "nonexistent99"})
	if err != nil {
		t.Fatalf("expected nil error deleting non-existent path, got: %v", err)
	}
}

func TestError_SetInvalidValue(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum90"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create the dummy interface first.
	if err := client.Set(ctx, basePath, nil); err != nil {
		t.Fatalf("Set create dummy: %v", err)
	}

	// Try setting a non-numeric MTU value. VyOS should reject this.
	err := client.Set(ctx, []string{"interfaces", "dummy", name, "mtu"}, "notanumber")
	if err == nil {
		t.Fatal("expected error setting non-numeric MTU, got nil")
	}
	t.Logf("Set invalid MTU: %v", err)
}

func TestError_BatchPartialInvalid(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum91"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Batch with one valid and one invalid operation.
	// VyOS should reject the entire batch atomically.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "valid"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "mtu"}, Value: "notanumber"},
	}
	err := client.BatchConfigure(ctx, ops)
	if err == nil {
		t.Fatal("expected error for batch with invalid operation, got nil")
	}
	t.Logf("Batch partial invalid: %v", err)

	// Verify the valid parts were also rolled back (atomic rejection).
	exists, err := client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists check after failed batch: %v", err)
	}
	if exists {
		t.Fatal("dummy interface exists after failed batch; expected atomic rollback")
	}
}
