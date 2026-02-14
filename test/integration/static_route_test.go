//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestStaticRoute_CreateReadDelete(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const prefix = "198.51.100.0/24"
	basePath := []string{"protocols", "static", "route", prefix}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create route with next-hop.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"protocols", "static", "route", prefix, "next-hop", "10.0.0.1"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create route: %v", err)
	}

	// Read back and verify next-hop.
	config := readStaticRouteConfig(ctx, t, client, prefix)
	nhRaw, ok := config["next-hop"]
	if !ok {
		t.Fatal("next-hop missing from route config")
	}

	var nhConfig map[string]json.RawMessage
	if err := json.Unmarshal(nhRaw, &nhConfig); err != nil {
		t.Fatalf("unmarshal next-hop: %v", err)
	}
	if _, ok := nhConfig["10.0.0.1"]; !ok {
		t.Fatalf("next-hop 10.0.0.1 not found, got keys: %v", configKeys(nhConfig))
	}

	// Delete and verify gone.
	if err := client.Delete(ctx, basePath); err != nil {
		t.Fatalf("Delete route: %v", err)
	}

	exists, err := client.Exists(ctx, basePath)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("route still exists after delete")
	}
}

func TestStaticRoute_UpdateNextHop(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const prefix = "203.0.113.0/24"
	basePath := []string{"protocols", "static", "route", prefix}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create with initial next-hop.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"protocols", "static", "route", prefix, "next-hop", "10.0.0.1"}},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	// Update: remove old next-hop, add new one.
	updateOps := []vyosclient.Operation{
		{Op: "delete", Path: []any{"protocols", "static", "route", prefix, "next-hop", "10.0.0.1"}},
		{Op: "set", Path: []any{"protocols", "static", "route", prefix, "next-hop", "10.0.0.2"}},
	}
	if err := client.BatchConfigure(ctx, updateOps); err != nil {
		t.Fatalf("BatchConfigure update: %v", err)
	}

	config := readStaticRouteConfig(ctx, t, client, prefix)
	nhRaw, ok := config["next-hop"]
	if !ok {
		t.Fatal("next-hop missing after update")
	}

	var nhConfig map[string]json.RawMessage
	if err := json.Unmarshal(nhRaw, &nhConfig); err != nil {
		t.Fatalf("unmarshal next-hop: %v", err)
	}
	if _, ok := nhConfig["10.0.0.2"]; !ok {
		t.Fatalf("next-hop 10.0.0.2 not found after update, got keys: %v", configKeys(nhConfig))
	}
	if _, ok := nhConfig["10.0.0.1"]; ok {
		t.Fatal("old next-hop 10.0.0.1 still present after update")
	}
}

func TestStaticRoute_Description(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	const prefix = "192.0.2.0/24"
	basePath := []string{"protocols", "static", "route", prefix}
	t.Cleanup(func() { deleteIfExists(t, client, basePath) })
	deleteIfExists(t, client, basePath)

	// Create route with description and next-hop.
	ops := []vyosclient.Operation{
		{Op: "set", Path: []any{"protocols", "static", "route", prefix, "next-hop", "10.0.0.1"}},
		{Op: "set", Path: []any{"protocols", "static", "route", prefix, "description"}, Value: "test route"},
	}
	if err := client.BatchConfigure(ctx, ops); err != nil {
		t.Fatalf("BatchConfigure create: %v", err)
	}

	config := readStaticRouteConfig(ctx, t, client, prefix)
	assertStringField(t, config, "description", "test route")

	// Update description.
	if err := client.Set(ctx, []string{"protocols", "static", "route", prefix, "description"}, "updated route"); err != nil {
		t.Fatalf("Set description: %v", err)
	}

	config = readStaticRouteConfig(ctx, t, client, prefix)
	assertStringField(t, config, "description", "updated route")
}

func readStaticRouteConfig(ctx context.Context, t *testing.T, client *vyosclient.Client, prefix string) map[string]json.RawMessage {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"protocols", "static", "route", prefix})
	if err != nil {
		t.Fatalf("ShowConfig protocols static route %s: %v", prefix, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal route %s config: %v", prefix, err)
	}
	return config
}
