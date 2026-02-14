package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
	"github.com/pulumi/pulumi-go-provider/infer"
)

func TestInterfaceEthernet_CreateDryRun(t *testing.T) {
	t.Parallel()
	resource := InterfaceEthernet{}

	resp, err := resource.Create(context.Background(), infer.CreateRequest[InterfaceEthernetArgs]{
		Name: "test-eth0",
		Inputs: InterfaceEthernetArgs{
			Name:      "eth0",
			Addresses: []string{"dhcp"},
		},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Create dry run error: %v", err)
	}
	if resp.ID != "eth0" {
		t.Errorf("ID = %q, want %q", resp.ID, "eth0")
	}
	if resp.Output.Name != "eth0" {
		t.Errorf("Name = %q, want %q", resp.Output.Name, "eth0")
	}
	if len(resp.Output.Addresses) != 1 || resp.Output.Addresses[0] != "dhcp" {
		t.Errorf("Addresses = %v, want [dhcp]", resp.Output.Addresses)
	}
}

func TestInterfaceEthernet_UpdateDryRun(t *testing.T) {
	t.Parallel()
	resource := InterfaceEthernet{}

	desc := "updated"
	resp, err := resource.Update(context.Background(), infer.UpdateRequest[InterfaceEthernetArgs, InterfaceEthernetState]{
		ID: "eth0",
		State: InterfaceEthernetState{InterfaceEthernetArgs: InterfaceEthernetArgs{
			Name:      "eth0",
			Addresses: []string{"dhcp"},
		}},
		Inputs: InterfaceEthernetArgs{
			Name:        "eth0",
			Addresses:   []string{"dhcp"},
			Description: &desc,
		},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Update dry run error: %v", err)
	}
	if resp.Output.Description == nil || *resp.Output.Description != "updated" {
		t.Errorf("Description = %v, want 'updated'", resp.Output.Description)
	}
}

func TestInterfaceEthernet_BuildCreateOps(t *testing.T) {
	t.Parallel()

	desc := "test interface"
	mtu := 9000
	disable := true

	ops := buildEthernetOps(InterfaceEthernetArgs{
		Name:        "eth0",
		Addresses:   []string{"dhcp", "10.0.0.1/24"},
		Description: &desc,
		MTU:         &mtu,
		Disable:     &disable,
	})

	// 2 addresses + description + disable + mtu = 5
	if len(ops) != 5 {
		t.Fatalf("got %d ops, want 5: %+v", len(ops), ops)
	}

	// Address ops have value in path, not in Value field.
	assertOp(t, ops[0], "set", []any{"interfaces", "ethernet", "eth0", "address", "dhcp"}, nil)
	assertOp(t, ops[1], "set", []any{"interfaces", "ethernet", "eth0", "address", "10.0.0.1/24"}, nil)

	// Description has value.
	assertOp(t, ops[2], "set", []any{"interfaces", "ethernet", "eth0", "description"}, "test interface")

	// Disable is valueless.
	assertOp(t, ops[3], "set", []any{"interfaces", "ethernet", "eth0", "disable"}, nil)

	// MTU value is a string.
	assertOp(t, ops[4], "set", []any{"interfaces", "ethernet", "eth0", "mtu"}, "9000")
}

func TestInterfaceEthernet_BuildCreateOpsAllFields(t *testing.T) {
	t.Parallel()

	desc := "uplink"
	mtu := 1500
	duplex := "full"
	speed := "1000"
	mac := "aa:bb:cc:dd:ee:ff"

	ops := buildEthernetOps(InterfaceEthernetArgs{
		Name:        "eth1",
		Addresses:   []string{"192.168.1.1/24"},
		Description: &desc,
		MTU:         &mtu,
		Duplex:      &duplex,
		Speed:       &speed,
		MAC:         &mac,
	})

	// 1 address + description + mtu + duplex + speed + mac = 6
	if len(ops) != 6 {
		t.Fatalf("got %d ops, want 6: %+v", len(ops), ops)
	}

	assertOp(t, ops[0], "set", []any{"interfaces", "ethernet", "eth1", "address", "192.168.1.1/24"}, nil)
	assertOp(t, ops[1], "set", []any{"interfaces", "ethernet", "eth1", "description"}, "uplink")
	assertOp(t, ops[2], "set", []any{"interfaces", "ethernet", "eth1", "mtu"}, "1500")
	assertOp(t, ops[3], "set", []any{"interfaces", "ethernet", "eth1", "duplex"}, "full")
	assertOp(t, ops[4], "set", []any{"interfaces", "ethernet", "eth1", "speed"}, "1000")
	assertOp(t, ops[5], "set", []any{"interfaces", "ethernet", "eth1", "mac"}, "aa:bb:cc:dd:ee:ff")
}

func TestInterfaceEthernet_BuildCreateOpsMinimal(t *testing.T) {
	t.Parallel()

	ops := buildEthernetOps(InterfaceEthernetArgs{Name: "eth0"})
	if len(ops) != 0 {
		t.Fatalf("got %d ops for no-op create, want 0: %+v", len(ops), ops)
	}
}

func TestInterfaceEthernet_BuildCreateOpsDisableFalse(t *testing.T) {
	t.Parallel()

	// Disable=false should NOT produce a set operation.
	disable := false
	ops := buildEthernetOps(InterfaceEthernetArgs{
		Name:    "eth0",
		Disable: &disable,
	})
	if len(ops) != 0 {
		t.Fatalf("disable=false should produce 0 ops, got %d: %+v", len(ops), ops)
	}
}

func TestInterfaceEthernet_UpdateAddressDiff(t *testing.T) {
	t.Parallel()

	ops := buildEthernetUpdateOps(
		InterfaceEthernetArgs{
			Name:      "eth0",
			Addresses: []string{"dhcp", "10.0.0.1/24"},
		},
		InterfaceEthernetArgs{
			Name:      "eth0",
			Addresses: []string{"dhcp", "10.0.0.2/24"},
		},
	)

	// Should delete 10.0.0.1/24 and add 10.0.0.2/24; dhcp stays.
	if len(ops) != 2 {
		t.Fatalf("got %d ops, want 2: %+v", len(ops), ops)
	}

	// Find delete and set ops (order may vary due to map iteration).
	var deleteOp, setOp *opRef
	for i := range ops {
		switch ops[i].Op {
		case "delete":
			deleteOp = &opRef{ops[i]}
		case "set":
			setOp = &opRef{ops[i]}
		}
	}

	if deleteOp == nil {
		t.Fatal("missing delete op for removed address")
	}
	assertOp(t, deleteOp.op, "delete", []any{"interfaces", "ethernet", "eth0", "address", "10.0.0.1/24"}, nil)

	if setOp == nil {
		t.Fatal("missing set op for added address")
	}
	assertOp(t, setOp.op, "set", []any{"interfaces", "ethernet", "eth0", "address", "10.0.0.2/24"}, nil)
}

func TestInterfaceEthernet_UpdateDisableToggle(t *testing.T) {
	t.Parallel()

	disable := true

	// Set disable.
	ops := buildEthernetUpdateOps(
		InterfaceEthernetArgs{Name: "eth0"},
		InterfaceEthernetArgs{Name: "eth0", Disable: &disable},
	)
	if len(ops) != 1 {
		t.Fatalf("set disable: got %d ops, want 1: %+v", len(ops), ops)
	}
	assertOp(t, ops[0], "set", []any{"interfaces", "ethernet", "eth0", "disable"}, nil)

	// Remove disable.
	ops = buildEthernetUpdateOps(
		InterfaceEthernetArgs{Name: "eth0", Disable: &disable},
		InterfaceEthernetArgs{Name: "eth0"},
	)
	if len(ops) != 1 {
		t.Fatalf("remove disable: got %d ops, want 1: %+v", len(ops), ops)
	}
	assertOp(t, ops[0], "delete", []any{"interfaces", "ethernet", "eth0", "disable"}, nil)
}

func TestInterfaceEthernet_UpdateRemoveField(t *testing.T) {
	t.Parallel()

	desc := "old"
	mtu := 9000

	// Remove description and mtu.
	ops := buildEthernetUpdateOps(
		InterfaceEthernetArgs{Name: "eth0", Description: &desc, MTU: &mtu},
		InterfaceEthernetArgs{Name: "eth0"},
	)
	if len(ops) != 2 {
		t.Fatalf("got %d ops, want 2: %+v", len(ops), ops)
	}
	assertOp(t, ops[0], "delete", []any{"interfaces", "ethernet", "eth0", "description"}, nil)
	assertOp(t, ops[1], "delete", []any{"interfaces", "ethernet", "eth0", "mtu"}, nil)
}

func TestInterfaceEthernet_UpdateNoChange(t *testing.T) {
	t.Parallel()

	desc := "same"
	args := InterfaceEthernetArgs{
		Name:        "eth0",
		Addresses:   []string{"dhcp"},
		Description: &desc,
	}

	ops := buildEthernetUpdateOps(args, args)
	if len(ops) != 0 {
		t.Fatalf("no-change update should produce 0 ops, got %d: %+v", len(ops), ops)
	}
}

func TestInterfaceEthernet_ParseConfig(t *testing.T) {
	t.Parallel()

	configJSON := json.RawMessage(`{
		"address": ["dhcp", "10.0.0.1/24"],
		"description": "uplink",
		"disable": {},
		"mtu": "1500",
		"duplex": "auto",
		"speed": "auto",
		"hw-id": "52:54:00:12:34:56",
		"mac": "aa:bb:cc:dd:ee:ff"
	}`)

	args, err := parseEthernetConfig("eth0", configJSON)
	if err != nil {
		t.Fatalf("parseEthernetConfig error: %v", err)
	}

	if args.Name != "eth0" {
		t.Errorf("Name = %q, want %q", args.Name, "eth0")
	}
	if !slices.Equal(args.Addresses, []string{"dhcp", "10.0.0.1/24"}) {
		t.Errorf("Addresses = %v, want [dhcp 10.0.0.1/24]", args.Addresses)
	}
	if args.Description == nil || *args.Description != "uplink" {
		t.Errorf("Description = %v, want 'uplink'", args.Description)
	}
	if args.Disable == nil || !*args.Disable {
		t.Error("Disable should be true (key present)")
	}
	if args.MTU == nil || *args.MTU != 1500 {
		t.Errorf("MTU = %v, want 1500", args.MTU)
	}
	if args.Duplex == nil || *args.Duplex != "auto" {
		t.Errorf("Duplex = %v, want 'auto'", args.Duplex)
	}
	if args.Speed == nil || *args.Speed != "auto" {
		t.Errorf("Speed = %v, want 'auto'", args.Speed)
	}
	if args.MAC == nil || *args.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC = %v, want 'aa:bb:cc:dd:ee:ff'", args.MAC)
	}
}

func TestInterfaceEthernet_ParseSingleAddress(t *testing.T) {
	t.Parallel()

	// VyOS returns a plain string when there is only one address.
	configJSON := json.RawMessage(`{"address": "dhcp"}`)

	args, err := parseEthernetConfig("eth0", configJSON)
	if err != nil {
		t.Fatalf("parseEthernetConfig error: %v", err)
	}
	if !slices.Equal(args.Addresses, []string{"dhcp"}) {
		t.Errorf("Addresses = %v, want [dhcp]", args.Addresses)
	}
}

func TestInterfaceEthernet_ParseNoAddress(t *testing.T) {
	t.Parallel()

	configJSON := json.RawMessage(`{"description": "empty"}`)

	args, err := parseEthernetConfig("eth0", configJSON)
	if err != nil {
		t.Fatalf("parseEthernetConfig error: %v", err)
	}
	if len(args.Addresses) != 0 {
		t.Errorf("Addresses = %v, want empty", args.Addresses)
	}
}

func TestInterfaceEthernet_ParseMTUAsNumber(t *testing.T) {
	t.Parallel()

	// VyOS might return mtu as a JSON number in some versions.
	configJSON := json.RawMessage(`{"mtu": 9000}`)

	args, err := parseEthernetConfig("eth0", configJSON)
	if err != nil {
		t.Fatalf("parseEthernetConfig error: %v", err)
	}
	if args.MTU == nil || *args.MTU != 9000 {
		t.Errorf("MTU = %v, want 9000", args.MTU)
	}
}

func TestInterfaceEthernet_ParseDisableAbsent(t *testing.T) {
	t.Parallel()

	// No "disable" key means interface is enabled.
	configJSON := json.RawMessage(`{"address": "dhcp"}`)

	args, err := parseEthernetConfig("eth0", configJSON)
	if err != nil {
		t.Fatalf("parseEthernetConfig error: %v", err)
	}
	if args.Disable != nil {
		t.Errorf("Disable = %v, want nil (absent means enabled)", args.Disable)
	}
}

// opRef wraps an operation to avoid taking address of range variable.
type opRef struct {
	op vyosclient.Operation
}

// assertOp verifies that an Operation matches the expected op, path, and value.
func assertOp(t *testing.T, op vyosclient.Operation, wantOp string, wantPath []any, wantValue any) {
	t.Helper()
	if op.Op != wantOp {
		t.Errorf("op = %q, want %q", op.Op, wantOp)
	}
	if len(op.Path) != len(wantPath) {
		t.Errorf("path length = %d, want %d: %v", len(op.Path), len(wantPath), op.Path)
		return
	}
	for i := range wantPath {
		if fmt.Sprint(op.Path[i]) != fmt.Sprint(wantPath[i]) {
			t.Errorf("path[%d] = %v, want %v", i, op.Path[i], wantPath[i])
		}
	}
	if wantValue == nil && op.Value != nil {
		t.Errorf("value = %v, want nil", op.Value)
	}
	if wantValue != nil && fmt.Sprint(op.Value) != fmt.Sprint(wantValue) {
		t.Errorf("value = %v, want %v", op.Value, wantValue)
	}
}
