//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

// These tests use firewall group address-group as a safe tagNode resource
// for testing create/read/update/delete lifecycle with string and multi-value
// fields. Address groups can be freely created and deleted without affecting
// VM connectivity.

func TestFirewallGroupAddressGroup_CreateReadDelete(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "integ-test"
	basePath := []string{"firewall", "group", "address-group", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create address group with description and addresses.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"firewall", "group", "address-group", name}},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "description"}, Value: "integration test group"},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "address", "192.0.2.1"}},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "address", "192.0.2.2"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Read back and verify.
	config := readAddressGroupConfig(ctx, t, client, name)

	assertStringField(t, config, "description", "integration test group")

	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 2 {
		t.Fatalf("expected 2 addresses, got %d: %v", len(addrs), addrs)
	}
	addrSet := map[string]bool{}
	for _, a := range addrs {
		addrSet[a] = true
	}
	if !addrSet["192.0.2.1"] || !addrSet["192.0.2.2"] {
		t.Fatalf("addresses = %v, want [192.0.2.1, 192.0.2.2]", addrs)
	}

	// Delete and verify gone.
	if err := client.Delete(ctx, basePath); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, err := client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("address group still exists after delete")
	}
}

func TestFirewallGroupAddressGroup_Update(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "integ-upd"
	basePath := []string{"firewall", "group", "address-group", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create initial group.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"firewall", "group", "address-group", name}},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "description"}, Value: "original"},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "address", "192.0.2.10"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Update: change description, add address, remove old address.
	updateOps := []vyosclient.Operation{
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "description"}, Value: "updated"},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "address", "192.0.2.20"}},
		{Op: "delete", Path: []any{"firewall", "group", "address-group", name, "address", "192.0.2.10"}},
	}
	if err := client.BatchConfigure(ctx, updateOps); err != nil {
		t.Fatalf("BatchConfigure update: %v", err)
	}

	config := readAddressGroupConfig(ctx, t, client, name)
	assertStringField(t, config, "description", "updated")

	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "192.0.2.20" {
		t.Fatalf("after update: addresses = %v, want [192.0.2.20]", addrs)
	}
}

func TestFirewallGroupAddressGroup_MultiValueAddRemove(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "integ-multi"
	basePath := []string{"firewall", "group", "address-group", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with a single address.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"firewall", "group", "address-group", name}},
		{Op: "set", Path: []any{"firewall", "group", "address-group", name, "address", "10.0.0.1"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Verify single value (VyOS returns string, not array, for single values).
	config := readAddressGroupConfig(ctx, t, client, name)
	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "10.0.0.1" {
		t.Fatalf("initial: addresses = %v, want [10.0.0.1]", addrs)
	}

	// Add a second address.
	if err := client.Set(ctx, []string{"firewall", "group", "address-group", name, "address", "10.0.0.2"}, nil); err != nil {
		t.Fatalf("Set second address: %v", err)
	}

	// Now VyOS should return an array.
	config = readAddressGroupConfig(ctx, t, client, name)
	addrs = parseAddresses(t, config["address"])
	if len(addrs) != 2 {
		t.Fatalf("after add: expected 2 addresses, got %d: %v", len(addrs), addrs)
	}

	// Remove the first address.
	if err := client.Delete(ctx, []string{"firewall", "group", "address-group", name, "address", "10.0.0.1"}); err != nil {
		t.Fatalf("Delete first address: %v", err)
	}

	// Back to single value.
	config = readAddressGroupConfig(ctx, t, client, name)
	addrs = parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "10.0.0.2" {
		t.Fatalf("after remove: addresses = %v, want [10.0.0.2]", addrs)
	}
}

// readAddressGroupConfig reads the full config for a firewall address group.
func readAddressGroupConfig(ctx context.Context, t *testing.T, client *vyosclient.Client, name string) map[string]json.RawMessage {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"firewall", "group", "address-group", name})
	if err != nil {
		t.Fatalf("ShowConfig firewall group address-group %s: %v", name, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal address-group %s config: %v", name, err)
	}
	return config
}
