//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestBatch_MixedSetDelete(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const name = "dum50"
	basePath := []string{"interfaces", "dummy", name}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with description and address.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "before"},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "address", "10.88.0.1/32"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Mixed batch: update description, delete address, add new address.
	mixedOps := []vyosclient.Operation{
		{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: "after"},
		{Op: "delete", Path: []any{"interfaces", "dummy", name, "address", "10.88.0.1/32"}},
		{Op: "set", Path: []any{"interfaces", "dummy", name, "address", "10.88.0.2/32"}},
	}
	if err := client.BatchConfigure(ctx, mixedOps); err != nil {
		t.Fatalf("BatchConfigure mixed: %v", err)
	}

	config := readDummyConfig(ctx, t, client, name)
	assertStringField(t, config, "description", "after")

	addrs := parseAddresses(t, config["address"])
	if len(addrs) != 1 || addrs[0] != "10.88.0.2/32" {
		t.Fatalf("addresses = %v, want [10.88.0.2/32]", addrs)
	}
}

func TestBatch_EmptyBatch(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	// Empty operation slice should not error.
	err := client.BatchConfigure(ctx, []vyosclient.Operation{})
	if err != nil {
		t.Fatalf("BatchConfigure with empty ops: %v", err)
	}
}

func TestBatch_LargeBatch(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const baseNum = 60
	const count = 12

	// Register cleanup for all dummy interfaces.
	for i := range count {
		name := fmt.Sprintf("dum%d", baseNum+i)
		path := []string{"interfaces", "dummy", name}
		t.Cleanup(func() { deleteIfExists(t, client, path) })
		deleteIfExists(t, client, path)
	}

	// Build a batch with 12 dummy interfaces, each with a description.
	var ops []vyosclient.Operation
	for i := range count {
		name := fmt.Sprintf("dum%d", baseNum+i)
		ops = append(ops,
			vyosclient.Operation{Op: "set", Path: []any{"interfaces", "dummy", name}},
			vyosclient.Operation{Op: "set", Path: []any{"interfaces", "dummy", name, "description"}, Value: fmt.Sprintf("large-batch-%d", i)},
		)
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure large batch (%d ops): %v", len(ops), err)
	}

	// Verify all interfaces were created.
	for i := range count {
		name := fmt.Sprintf("dum%d", baseNum+i)
		config := readDummyConfig(ctx, t, client, name)
		assertStringField(t, config, "description", fmt.Sprintf("large-batch-%d", i))
	}
}
