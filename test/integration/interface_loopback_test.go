//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

// VyOS only allows loopback named "lo", which exists at the OS level but may
// have an empty config. These tests set fields on lo and clean up after.

func TestInterfaceLoopback_DescriptionAndAddress(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "lo"
	const testAddr = "10.88.0.1/32"

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_ = client.Delete(cleanupCtx, []string{"interfaces", "loopback", name, "address", testAddr})
		_ = client.Delete(cleanupCtx, []string{"interfaces", "loopback", name, "description"})
	})

	// Set description and add address using BatchConfigure.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "loopback", name, "description"}, Value: "test-lo"},
		{Op: "set", Path: []any{"interfaces", "loopback", name, "address", testAddr}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure: %v", err)
	}

	// Read back and verify.
	config := readLoopbackConfig(ctx, t, client, name)

	assertStringField(t, config, "description", "test-lo")

	addrs := parseAddresses(t, config["address"])
	found := false
	for _, a := range addrs {
		if a == testAddr {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("address %s not found in %v", testAddr, addrs)
	}
}

func TestInterfaceLoopback_Update(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "lo"

	t.Cleanup(func() {
		_ = client.Delete(context.Background(), []string{"interfaces", "loopback", name, "description"})
	})

	// Set initial description.
	if err := client.Set(ctx, []string{"interfaces", "loopback", name, "description"}, "initial"); err != nil {
		t.Fatalf("Set description: %v", err)
	}

	config := readLoopbackConfig(ctx, t, client, name)
	assertStringField(t, config, "description", "initial")

	// Update description.
	if err := client.Set(ctx, []string{"interfaces", "loopback", name, "description"}, "updated"); err != nil {
		t.Fatalf("Update description: %v", err)
	}

	config = readLoopbackConfig(ctx, t, client, name)
	assertStringField(t, config, "description", "updated")
}

// readLoopbackConfig reads the full config for a loopback interface.
func readLoopbackConfig(ctx context.Context, t *testing.T, client *vyosclient.Client, name string) map[string]json.RawMessage {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"interfaces", "loopback", name})
	if err != nil {
		t.Fatalf("ShowConfig interfaces loopback %s: %v", name, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal loopback %s config: %v", name, err)
	}
	return config
}
