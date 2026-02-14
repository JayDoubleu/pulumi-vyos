//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestInterfaceEthernet_ReadExisting(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	data, err := client.ShowConfig(ctx, []string{"interfaces", "ethernet", "eth0"})
	if err != nil {
		t.Fatalf("ShowConfig interfaces ethernet eth0: %v", err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal eth0 config: %v", err)
	}

	if _, ok := config["address"]; !ok {
		t.Error("eth0 config missing 'address' field")
	}
	if _, ok := config["hw-id"]; !ok {
		t.Error("eth0 config missing 'hw-id' field")
	}

	t.Logf("eth0 config keys: %v", configKeys(config))
}

func TestInterfaceEthernet_Description(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	// Save the original description so we can restore it.
	origDesc := readEthField(t, client, ctx, "eth0", "description")
	t.Cleanup(func() {
		if origDesc == "" {
			_ = client.Delete(ctx, []string{"interfaces", "ethernet", "eth0", "description"})
		} else {
			_ = client.Set(ctx, []string{"interfaces", "ethernet", "eth0", "description"}, origDesc)
		}
	})

	// Set a description.
	const desc = "integ-test-desc"
	if err := client.Set(ctx, []string{"interfaces", "ethernet", "eth0", "description"}, desc); err != nil {
		t.Fatalf("Set description: %v", err)
	}

	got := readEthField(t, client, ctx, "eth0", "description")
	if got != desc {
		t.Fatalf("after set: description = %q, want %q", got, desc)
	}

	// Update the description.
	const desc2 = "integ-test-desc-updated"
	if err := client.Set(ctx, []string{"interfaces", "ethernet", "eth0", "description"}, desc2); err != nil {
		t.Fatalf("Update description: %v", err)
	}

	got = readEthField(t, client, ctx, "eth0", "description")
	if got != desc2 {
		t.Fatalf("after update: description = %q, want %q", got, desc2)
	}
}

func TestInterfaceEthernet_MTU(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	origMTU := readEthField(t, client, ctx, "eth0", "mtu")
	t.Cleanup(func() {
		if origMTU == "" {
			_ = client.Delete(ctx, []string{"interfaces", "ethernet", "eth0", "mtu"})
		} else {
			_ = client.Set(ctx, []string{"interfaces", "ethernet", "eth0", "mtu"}, origMTU)
		}
	})

	// Set MTU to a safe value. Keep it reasonable to avoid connectivity issues.
	const mtu = "1400"
	if err := client.Set(ctx, []string{"interfaces", "ethernet", "eth0", "mtu"}, mtu); err != nil {
		t.Fatalf("Set mtu: %v", err)
	}

	got := readEthField(t, client, ctx, "eth0", "mtu")
	if got != mtu {
		t.Fatalf("after set: mtu = %q, want %q", got, mtu)
	}
}

func TestInterfaceEthernet_AddStaticAddress(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	// Add a static address alongside whatever is already configured.
	// Use a link-local-ish address to avoid routing conflicts.
	const addr = "198.51.100.1/32"

	t.Cleanup(func() {
		_ = client.Delete(ctx, []string{"interfaces", "ethernet", "eth0", "address", addr})
	})

	// Add the address using batch to exercise that code path.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "ethernet", "eth0", "address", addr}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure add address: %v", err)
	}

	// Read back and verify the address is present.
	data, err := client.ShowConfig(ctx, []string{"interfaces", "ethernet", "eth0"})
	if err != nil {
		t.Fatalf("ShowConfig: %v", err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	addrs := parseAddresses(t, config["address"])
	found := false
	for _, a := range addrs {
		if a == addr {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("address %q not found in %v", addr, addrs)
	}
}

// readEthField reads a single string field from eth0 config.
// Returns empty string if the field is absent.
func readEthField(t *testing.T, client *vyosclient.Client, ctx context.Context, iface, field string) string {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"interfaces", "ethernet", iface})
	if err != nil {
		t.Fatalf("ShowConfig interfaces ethernet %s: %v", iface, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal %s config: %v", iface, err)
	}

	raw, ok := config[field]
	if !ok {
		return ""
	}

	var val string
	if err := json.Unmarshal(raw, &val); err != nil {
		t.Fatalf("unmarshal %s.%s: %v", iface, field, err)
	}
	return val
}

// parseAddresses handles VyOS multi-value address field (string or array).
func parseAddresses(t *testing.T, data json.RawMessage) []string {
	t.Helper()

	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		t.Fatalf("parse addresses: not string or array: %s", string(data))
	}
	return []string{single}
}

// configKeys returns the keys of a map for logging.
func configKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
