//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestNATSourceRule_CreateReadDelete(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const rule = "100"
	basePath := []string{"nat", "source", "rule", rule}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create NAT source rule with outbound interface, masquerade, and source address.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"nat", "source", "rule", rule, "outbound-interface", "name"}, Value: "eth0"},
		{Op: "set", Path: []any{"nat", "source", "rule", rule, "translation", "address"}, Value: "masquerade"},
		{Op: "set", Path: []any{"nat", "source", "rule", rule, "source", "address"}, Value: "10.0.0.0/24"},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create NAT rule: %v", err)
	}

	// Read back and verify.
	config := readNATSourceRuleConfig(ctx, t, client, rule)

	// Check outbound-interface -> name
	oifRaw, ok := config["outbound-interface"]
	if !ok {
		t.Fatal("outbound-interface missing")
	}
	var oifConfig map[string]json.RawMessage
	if err := json.Unmarshal(oifRaw, &oifConfig); err != nil {
		t.Fatalf("unmarshal outbound-interface: %v", err)
	}
	assertStringField(t, oifConfig, "name", "eth0")

	// Check translation -> address
	trRaw, ok := config["translation"]
	if !ok {
		t.Fatal("translation missing")
	}
	var trConfig map[string]json.RawMessage
	if err := json.Unmarshal(trRaw, &trConfig); err != nil {
		t.Fatalf("unmarshal translation: %v", err)
	}
	assertStringField(t, trConfig, "address", "masquerade")

	// Check source -> address
	srcRaw, ok := config["source"]
	if !ok {
		t.Fatal("source missing")
	}
	var srcConfig map[string]json.RawMessage
	if err := json.Unmarshal(srcRaw, &srcConfig); err != nil {
		t.Fatalf("unmarshal source: %v", err)
	}
	assertStringField(t, srcConfig, "address", "10.0.0.0/24")

	// Delete and verify gone.
	if err := client.Delete(ctx, basePath); err != nil {
		t.Fatalf("Delete NAT rule: %v", err)
	}

	exists, err := client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("NAT rule still exists after delete")
	}
}

func TestNATSourceRule_Update(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const rule = "101"
	basePath := []string{"nat", "source", "rule", rule}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with initial source address.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"nat", "source", "rule", rule, "outbound-interface", "name"}, Value: "eth0"},
		{Op: "set", Path: []any{"nat", "source", "rule", rule, "translation", "address"}, Value: "masquerade"},
		{Op: "set", Path: []any{"nat", "source", "rule", rule, "source", "address"}, Value: "10.0.0.0/24"},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Update source address.
	if err := client.Set(ctx, []string{"nat", "source", "rule", rule, "source", "address"}, "10.0.1.0/24"); err != nil {
		t.Fatalf("Set source address: %v", err)
	}

	config := readNATSourceRuleConfig(ctx, t, client, rule)
	srcRaw, ok := config["source"]
	if !ok {
		t.Fatal("source missing after update")
	}
	var srcConfig map[string]json.RawMessage
	if err := json.Unmarshal(srcRaw, &srcConfig); err != nil {
		t.Fatalf("unmarshal source: %v", err)
	}
	assertStringField(t, srcConfig, "address", "10.0.1.0/24")
}

func readNATSourceRuleConfig(ctx context.Context, t *testing.T, client *vyosclient.Client, rule string) map[string]json.RawMessage {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"nat", "source", "rule", rule})
	if err != nil {
		t.Fatalf("ShowConfig nat source rule %s: %v", rule, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal nat source rule %s config: %v", rule, err)
	}
	return config
}
