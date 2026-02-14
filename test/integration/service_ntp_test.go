//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestServiceNTP_AddRemoveServer(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const server = "198.51.100.1"
	basePath := []string{"service", "ntp", "server", server}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Add NTP server.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"service", "ntp", "server", server}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure add NTP server: %v", err)
	}

	// Verify it exists.
	exists, err := client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists NTP server: %v", err)
	}
	if !exists {
		t.Fatal("NTP server not found after add")
	}

	// Delete and verify gone.
	if err := client.Delete(ctx, basePath); err != nil {
		t.Fatalf("Delete NTP server: %v", err)
	}

	exists, err = client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("NTP server still exists after delete")
	}
}

func TestServiceNTP_ServerWithPool(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const server = "198.51.100.2"
	basePath := []string{"service", "ntp", "server", server}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Add NTP server with pool (valueless bool).
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"service", "ntp", "server", server}},
		{Op: "set", Path: []any{"service", "ntp", "server", server, "pool"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure add NTP server with pool: %v", err)
	}

	// Read back and verify pool is present.
	config := readNTPServerConfig(ctx, t, client, server)
	if _, ok := config["pool"]; !ok {
		t.Fatal("pool field missing after set")
	}

	// Remove the pool flag.
	if err := client.Delete(ctx, []string{"service", "ntp", "server", server, "pool"}); err != nil {
		t.Fatalf("Delete pool: %v", err)
	}

	// After removing pool (the only attribute), the server node may exist
	// but have empty config. Use Exists on the pool path directly to
	// verify it was removed, since ShowConfig fails on empty nodes.
	poolExists, err := client.Exists(ctx, []string{"service", "ntp", "server", server, "pool"})
	if err != nil {
		t.Fatalf("Exists pool after delete: %v", err)
	}
	if poolExists {
		t.Fatal("pool still exists after delete")
	}
}

func readNTPServerConfig(ctx context.Context, t *testing.T, client *vyosclient.Client, server string) map[string]json.RawMessage {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"service", "ntp", "server", server})
	if err != nil {
		t.Fatalf("ShowConfig service ntp server %s: %v", server, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal ntp server %s config: %v", server, err)
	}
	return config
}
