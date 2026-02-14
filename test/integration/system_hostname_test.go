//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
)

func TestSystemHostname_CRUD(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	// Read the current hostname so we can restore it after the test.
	original := readHostname(ctx, t, client)
	t.Cleanup(func() {
		if err := client.Set(ctx, []string{"system", "host-name"}, original); err != nil {
			t.Errorf("cleanup: failed to restore hostname to %q: %v", original, err)
		}
	})

	// Create: set a new hostname.
	if err := client.Set(ctx, []string{"system", "host-name"}, "integ-test-1"); err != nil {
		t.Fatalf("Set integ-test-1: %v", err)
	}
	if got := readHostname(ctx, t, client); got != "integ-test-1" {
		t.Fatalf("after create: got hostname %q, want %q", got, "integ-test-1")
	}

	// Update: change to a different hostname.
	if err := client.Set(ctx, []string{"system", "host-name"}, "integ-test-2"); err != nil {
		t.Fatalf("Set integ-test-2: %v", err)
	}
	if got := readHostname(ctx, t, client); got != "integ-test-2" {
		t.Fatalf("after update: got hostname %q, want %q", got, "integ-test-2")
	}
}

func TestSystemHostname_ReadExisting(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	data, err := client.ShowConfig(ctx, []string{"system"})
	if err != nil {
		t.Fatalf("ShowConfig system: %v", err)
	}

	var sysConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &sysConfig); err != nil {
		t.Fatalf("unmarshal system config: %v", err)
	}

	raw, ok := sysConfig["host-name"]
	if !ok {
		t.Fatal("system config has no host-name field")
	}

	var hostname string
	if err := json.Unmarshal(raw, &hostname); err != nil {
		t.Fatalf("unmarshal host-name: %v", err)
	}
	if hostname == "" {
		t.Fatal("host-name is empty")
	}
	t.Logf("current hostname: %s", hostname)
}

func TestSystemHostname_Idempotent(t *testing.T) {
	client := vyosClient(t)
	ctx := context.Background()

	original := readHostname(ctx, t, client)
	t.Cleanup(func() {
		if err := client.Set(ctx, []string{"system", "host-name"}, original); err != nil {
			t.Errorf("cleanup: failed to restore hostname to %q: %v", original, err)
		}
	})

	const name = "integ-idempotent"

	// Set once.
	if err := client.Set(ctx, []string{"system", "host-name"}, name); err != nil {
		t.Fatalf("first Set: %v", err)
	}

	// Set again with the same value; should succeed without error.
	if err := client.Set(ctx, []string{"system", "host-name"}, name); err != nil {
		t.Fatalf("second Set (idempotent): %v", err)
	}

	if got := readHostname(ctx, t, client); got != name {
		t.Fatalf("after idempotent set: got %q, want %q", got, name)
	}
}

// readHostname fetches the current system hostname from VyOS.
func readHostname(ctx context.Context, t *testing.T, client *vyosclient.Client) string {
	t.Helper()

	data, err := client.ShowConfig(ctx, []string{"system"})
	if err != nil {
		t.Fatalf("ShowConfig system: %v", err)
	}

	var sysConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &sysConfig); err != nil {
		t.Fatalf("unmarshal system config: %v", err)
	}

	raw, ok := sysConfig["host-name"]
	if !ok {
		t.Fatal("system config missing host-name")
	}

	var hostname string
	if err := json.Unmarshal(raw, &hostname); err != nil {
		t.Fatalf("unmarshal host-name: %v", err)
	}
	return hostname
}
