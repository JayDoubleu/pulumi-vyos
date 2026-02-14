//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestInterfaceDummy_CreateReadDelete(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum0"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })

	// Delete any leftover from a previous failed run.
	deleteIfExists(t, client, basePath)

	// Create dum0 with all four field types.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "test-dummy"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "mtu"}, Value: "1400"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "address", "10.99.0.1/32"}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "disable"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create dum0: %v", err)
	}

	// Read back and verify all fields.
	data, err := client.ShowConfig(ctx, basePath)
	if err != nil {
		t.Fatalf("ShowConfig dum0: %v", err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal dum0 config: %v", err)
	}

	// String field: description
	assertStringField(t, config, "description", "test-dummy")

	// Int field: mtu
	assertStringField(t, config, "mtu", "1400")

	// Multi field: address (single value returns as string)
	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "10.99.0.1/32" {
		t.Fatalf("address = %v, want [10.99.0.1/32]", addrs)
	}

	// Bool (valueless) field: disable should be present as empty object
	if _, ok := config["disable"]; !ok {
		t.Fatal("disable field missing after create")
	}

	// Delete and verify gone.
	if err := client.Delete(ctx, basePath); err != nil {
		t.Fatalf("Delete dum0: %v", err)
	}

	exists, err := client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("dum0 still exists after delete")
	}
}

func TestInterfaceDummy_Update(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum1"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with initial values.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "v1"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "mtu"}, Value: "1400"},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create dum1: %v", err)
	}

	// Update: change description, change mtu, add address.
	updateOps := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "v2"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "mtu"}, Value: "9000"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "address", "10.99.1.1/32"}},
	}
	if err := client.BatchConfigure(ctx, updateOps); err != nil {
		t.Fatalf("BatchConfigure update dum1: %v", err)
	}

	// Read back and verify updated values.
	data, err := client.ShowConfig(ctx, basePath)
	if err != nil {
		t.Fatalf("ShowConfig dum1: %v", err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal dum1 config: %v", err)
	}

	assertStringField(t, config, "description", "v2")
	assertStringField(t, config, "mtu", "9000")

	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "10.99.1.1/32" {
		t.Fatalf("address = %v, want [10.99.1.1/32]", addrs)
	}
}

func TestInterfaceDummy_BoolField(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum0"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create without disable.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "bool-test"},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Verify disable is absent.
	config := readDummyConfig(ctx, t, client, name)
	if _, ok := config["disable"]; ok {
		t.Fatal("disable should be absent initially")
	}

	// Set disable (valueless boolean).
	if err := client.Set(ctx, []string{"interfaces", "dummy", name, "disable"}, nil); err != nil {
		t.Fatalf("Set disable: %v", err)
	}

	// Verify disable is present.
	config = readDummyConfig(ctx, t, client, name)
	if _, ok := config["disable"]; !ok {
		t.Fatal("disable should be present after set")
	}

	// Remove disable.
	if err := client.Delete(ctx, []string{"interfaces", "dummy", name, "disable"}); err != nil {
		t.Fatalf("Delete disable: %v", err)
	}

	// Verify disable is gone again.
	config = readDummyConfig(ctx, t, client, name)
	if _, ok := config["disable"]; ok {
		t.Fatal("disable should be absent after delete")
	}
}

func TestInterfaceDummy_MultiField(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum0"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with two addresses.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "address", "10.99.0.1/32"}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "address", "10.99.0.2/32"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create with two addresses: %v", err)
	}

	// Read back and verify both addresses.
	config := readDummyConfig(ctx, t, client, name)
	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 2 {
		t.Fatalf("expected 2 addresses, got %d: %v", len(addrs), addrs)
	}

	addrSet := map[string]bool{}
	for _, a := range addrs {
		addrSet[a] = true
	}
	if !addrSet["10.99.0.1/32"] || !addrSet["10.99.0.2/32"] {
		t.Fatalf("addresses = %v, want [10.99.0.1/32, 10.99.0.2/32]", addrs)
	}

	// Remove one address.
	if err := client.Delete(ctx, []string{"interfaces", "dummy", name, "address", "10.99.0.2/32"}); err != nil {
		t.Fatalf("Delete address: %v", err)
	}

	config = readDummyConfig(ctx, t, client, name)
	addrs = parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "10.99.0.1/32" {
		t.Fatalf("after remove: addresses = %v, want [10.99.0.1/32]", addrs)
	}
}

func TestInterfaceDummy_NestedField(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum0"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with a nested field (ip source-validation).
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "ip", "source-validation"}, Value: "strict"},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create with nested field: %v", err)
	}

	// Read back and verify nested value.
	config := readDummyConfig(ctx, t, client, name)
	ipRaw, ok := config["ip"]
	if !ok {
		t.Fatal("ip subtree missing")
	}

	var ipConfig map[string]json.RawMessage
	if err := json.Unmarshal(ipRaw, &ipConfig); err != nil {
		t.Fatalf("unmarshal ip config: %v", err)
	}

	var sv string
	if err := json.Unmarshal(ipConfig["source-validation"], &sv); err != nil {
		t.Fatalf("unmarshal source-validation: %v", err)
	}
	if sv != "strict" {
		t.Fatalf("ip source-validation = %q, want %q", sv, "strict")
	}
}

// readDummyConfig reads the full config for a dummy interface.
func readDummyConfig(ctx context.Context, t *testing.T, client *vyosclient.Client, name string) map[string]json.RawMessage {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"interfaces", "dummy", name})
	if err != nil {
		t.Fatalf("ShowConfig interfaces dummy %s: %v", name, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal dummy %s config: %v", name, err)
	}
	return config
}

// assertStringField verifies a string field in a VyOS config map.
// VyOS may return numeric values as either strings or numbers, so this
// handles both JSON representations.
func assertStringField(t *testing.T, config map[string]json.RawMessage, field, want string) {
	t.Helper()

	raw, ok := config[field]
	if !ok {
		t.Fatalf("field %q missing", field)
	}

	// Try as string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s != want {
			t.Fatalf("%s = %q, want %q", field, s, want)
		}
		return
	}

	// Try as number (VyOS sometimes returns numeric values unquoted).
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		if n.String() != want {
			t.Fatalf("%s = %s, want %s", field, n.String(), want)
		}
		return
	}

	t.Fatalf("field %q: cannot parse %s as string or number", field, string(raw))
}
