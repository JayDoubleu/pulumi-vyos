package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// InterfaceEthernet manages a VyOS ethernet interface configuration.
type InterfaceEthernet struct{}

// InterfaceEthernetArgs defines the inputs for the InterfaceEthernet resource.
type InterfaceEthernetArgs struct {
	Name        string   `pulumi:"name"`
	Addresses   []string `pulumi:"addresses,optional"`
	Description *string  `pulumi:"description,optional"`
	Disable     *bool    `pulumi:"disable,optional"`
	MTU         *int     `pulumi:"mtu,optional"`
	Duplex      *string  `pulumi:"duplex,optional"`
	Speed       *string  `pulumi:"speed,optional"`
	MAC         *string  `pulumi:"mac,optional"`
}

// InterfaceEthernetState defines the outputs (persisted state) for InterfaceEthernet.
type InterfaceEthernetState struct {
	InterfaceEthernetArgs
}

// Annotate provides schema metadata for the resource.
func (*InterfaceEthernet) Annotate(a infer.Annotator) {
	a.Describe(new(InterfaceEthernet), "Manages a VyOS ethernet interface.")
}

// Annotate provides schema metadata for the inputs.
func (args *InterfaceEthernetArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Name, "The ethernet interface name (e.g. eth0).")
	a.Describe(&args.Addresses, "IP addresses or 'dhcp'/'dhcpv6' assigned to the interface.")
	a.Describe(&args.Description, "A description for the interface.")
	a.Describe(&args.Disable, "Disable the interface. True means the interface is administratively down.")
	a.Describe(&args.MTU, "Maximum transmission unit (68-16000).")
	a.Describe(&args.Duplex, "Duplex mode: auto, half, or full.")
	a.Describe(&args.Speed, "Link speed: auto, 10, 100, 1000, 2500, 5000, 10000, etc.")
	a.Describe(&args.MAC, "Override MAC address for the interface.")
}

// ethBasePath returns the VyOS config path prefix for an ethernet interface.
func ethBasePath(name string) []any {
	return []any{"interfaces", "ethernet", name}
}

// Create sets up an ethernet interface on VyOS.
func (InterfaceEthernet) Create(
	ctx context.Context,
	req infer.CreateRequest[InterfaceEthernetArgs],
) (infer.CreateResponse[InterfaceEthernetState], error) {
	if req.DryRun {
		return infer.CreateResponse[InterfaceEthernetState]{
			ID:     req.Inputs.Name,
			Output: InterfaceEthernetState{InterfaceEthernetArgs: req.Inputs},
		}, nil
	}

	ops := buildEthernetOps(req.Inputs)
	if len(ops) > 0 {
		client := getClient(ctx)
		if err := client.BatchConfigure(ctx, ops); err != nil {
			return infer.CreateResponse[InterfaceEthernetState]{},
				fmt.Errorf("configure interface ethernet %s: %w", req.Inputs.Name, err)
		}
	}

	return infer.CreateResponse[InterfaceEthernetState]{
		ID:     req.Inputs.Name,
		Output: InterfaceEthernetState{InterfaceEthernetArgs: req.Inputs},
	}, nil
}

// Read fetches the current ethernet interface config from VyOS.
func (InterfaceEthernet) Read(
	ctx context.Context,
	req infer.ReadRequest[InterfaceEthernetArgs, InterfaceEthernetState],
) (infer.ReadResponse[InterfaceEthernetArgs, InterfaceEthernetState], error) {
	client := getClient(ctx)
	name := req.State.Name

	data, err := client.ShowConfig(ctx, []string{"interfaces", "ethernet", name})
	if err != nil {
		return infer.ReadResponse[InterfaceEthernetArgs, InterfaceEthernetState]{},
			fmt.Errorf("read interface ethernet %s: %w", name, err)
	}

	args, err := parseEthernetConfig(name, data)
	if err != nil {
		return infer.ReadResponse[InterfaceEthernetArgs, InterfaceEthernetState]{},
			fmt.Errorf("parse interface ethernet %s config: %w", name, err)
	}

	state := InterfaceEthernetState{InterfaceEthernetArgs: args}
	return infer.ReadResponse[InterfaceEthernetArgs, InterfaceEthernetState]{
		ID:     name,
		Inputs: args,
		State:  state,
	}, nil
}

// Update modifies an existing ethernet interface configuration.
func (InterfaceEthernet) Update(
	ctx context.Context,
	req infer.UpdateRequest[InterfaceEthernetArgs, InterfaceEthernetState],
) (infer.UpdateResponse[InterfaceEthernetState], error) {
	if req.DryRun {
		return infer.UpdateResponse[InterfaceEthernetState]{
			Output: InterfaceEthernetState{InterfaceEthernetArgs: req.Inputs},
		}, nil
	}

	ops := buildEthernetUpdateOps(req.State.InterfaceEthernetArgs, req.Inputs)
	if len(ops) > 0 {
		client := getClient(ctx)
		if err := client.BatchConfigure(ctx, ops); err != nil {
			return infer.UpdateResponse[InterfaceEthernetState]{},
				fmt.Errorf("update interface ethernet %s: %w", req.Inputs.Name, err)
		}
	}

	return infer.UpdateResponse[InterfaceEthernetState]{
		Output: InterfaceEthernetState{InterfaceEthernetArgs: req.Inputs},
	}, nil
}

// Delete removes all managed configuration from the ethernet interface.
func (InterfaceEthernet) Delete(
	ctx context.Context,
	req infer.DeleteRequest[InterfaceEthernetState],
) (infer.DeleteResponse, error) {
	client := getClient(ctx)
	name := req.State.Name
	if err := client.Delete(ctx, []string{"interfaces", "ethernet", name}); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("delete interface ethernet %s: %w", name, err)
	}
	return infer.DeleteResponse{}, nil
}

// buildEthernetOps produces set operations for all non-nil fields in args.
func buildEthernetOps(args InterfaceEthernetArgs) []vyosclient.Operation {
	var ops []vyosclient.Operation
	base := ethBasePath(args.Name)

	// Multi-value addresses: each address is a value in the path, not a separate value field.
	for _, addr := range args.Addresses {
		ops = append(ops, vyosclient.Operation{
			Op:   "set",
			Path: append(append([]any{}, base...), "address", addr),
		})
	}

	if args.Description != nil {
		ops = append(ops, vyosclient.Operation{
			Op:    "set",
			Path:  append(append([]any{}, base...), "description"),
			Value: *args.Description,
		})
	}

	if args.Disable != nil && *args.Disable {
		ops = append(ops, vyosclient.Operation{
			Op:   "set",
			Path: append(append([]any{}, base...), "disable"),
		})
	}

	if args.MTU != nil {
		ops = append(ops, vyosclient.Operation{
			Op:    "set",
			Path:  append(append([]any{}, base...), "mtu"),
			Value: strconv.Itoa(*args.MTU),
		})
	}

	if args.Duplex != nil {
		ops = append(ops, vyosclient.Operation{
			Op:    "set",
			Path:  append(append([]any{}, base...), "duplex"),
			Value: *args.Duplex,
		})
	}

	if args.Speed != nil {
		ops = append(ops, vyosclient.Operation{
			Op:    "set",
			Path:  append(append([]any{}, base...), "speed"),
			Value: *args.Speed,
		})
	}

	if args.MAC != nil {
		ops = append(ops, vyosclient.Operation{
			Op:    "set",
			Path:  append(append([]any{}, base...), "mac"),
			Value: *args.MAC,
		})
	}

	return ops
}

// buildEthernetUpdateOps computes the diff between prev and cur state,
// producing set/delete operations for changed fields.
func buildEthernetUpdateOps(prev, cur InterfaceEthernetArgs) []vyosclient.Operation {
	var ops []vyosclient.Operation
	base := ethBasePath(cur.Name)

	// Address diff: delete removed, add added.
	prevAddrs := toSet(prev.Addresses)
	curAddrs := toSet(cur.Addresses)

	for addr := range prevAddrs {
		if !curAddrs[addr] {
			ops = append(ops, vyosclient.Operation{
				Op:   "delete",
				Path: append(append([]any{}, base...), "address", addr),
			})
		}
	}
	for addr := range curAddrs {
		if !prevAddrs[addr] {
			ops = append(ops, vyosclient.Operation{
				Op:   "set",
				Path: append(append([]any{}, base...), "address", addr),
			})
		}
	}

	// Description
	ops = append(ops, diffStringField(base, "description", prev.Description, cur.Description)...)

	// Disable (valueless boolean)
	ops = append(ops, diffDisableField(base, prev.Disable, cur.Disable)...)

	// MTU
	ops = append(ops, diffIntField(base, "mtu", prev.MTU, cur.MTU)...)

	// Duplex
	ops = append(ops, diffStringField(base, "duplex", prev.Duplex, cur.Duplex)...)

	// Speed
	ops = append(ops, diffStringField(base, "speed", prev.Speed, cur.Speed)...)

	// MAC
	ops = append(ops, diffStringField(base, "mac", prev.MAC, cur.MAC)...)

	return ops
}

// diffStringField returns operations for a single-value string field change.
func diffStringField(base []any, field string, prev, cur *string) []vyosclient.Operation {
	if ptrStr(prev) == ptrStr(cur) {
		return nil
	}
	path := append(append([]any{}, base...), field)
	if cur == nil {
		return []vyosclient.Operation{{Op: "delete", Path: path}}
	}
	return []vyosclient.Operation{{Op: "set", Path: path, Value: *cur}}
}

// diffIntField returns operations for a single-value int field change.
func diffIntField(base []any, field string, prev, cur *int) []vyosclient.Operation {
	if ptrInt(prev) == ptrInt(cur) {
		return nil
	}
	path := append(append([]any{}, base...), field)
	if cur == nil {
		return []vyosclient.Operation{{Op: "delete", Path: path}}
	}
	return []vyosclient.Operation{{Op: "set", Path: path, Value: strconv.Itoa(*cur)}}
}

// diffDisableField handles the valueless boolean "disable" field.
func diffDisableField(base []any, prev, cur *bool) []vyosclient.Operation {
	oldDisabled := prev != nil && *prev
	newDisabled := cur != nil && *cur
	if oldDisabled == newDisabled {
		return nil
	}
	path := append(append([]any{}, base...), "disable")
	if newDisabled {
		return []vyosclient.Operation{{Op: "set", Path: path}}
	}
	return []vyosclient.Operation{{Op: "delete", Path: path}}
}

// parseEthernetConfig unmarshals VyOS ShowConfig JSON into InterfaceEthernetArgs.
func parseEthernetConfig(name string, data json.RawMessage) (InterfaceEthernetArgs, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return InterfaceEthernetArgs{}, fmt.Errorf("unmarshal config: %w", err)
	}

	args := InterfaceEthernetArgs{Name: name}

	// Addresses: VyOS returns a string for single, array for multiple.
	if v, ok := raw["address"]; ok {
		addrs, err := parseMultiValue(v)
		if err != nil {
			return InterfaceEthernetArgs{}, fmt.Errorf("parse addresses: %w", err)
		}
		args.Addresses = addrs
	}

	if v, ok := raw["description"]; ok {
		var desc string
		if err := json.Unmarshal(v, &desc); err != nil {
			return InterfaceEthernetArgs{}, fmt.Errorf("parse description: %w", err)
		}
		args.Description = &desc
	}

	// Disable is a valueless boolean: presence of key means true.
	// VyOS returns it as an empty object {}.
	if _, ok := raw["disable"]; ok {
		t := true
		args.Disable = &t
	}

	if v, ok := raw["mtu"]; ok {
		mtu, err := parseIntFromJSON(v)
		if err != nil {
			return InterfaceEthernetArgs{}, fmt.Errorf("parse mtu: %w", err)
		}
		args.MTU = &mtu
	}

	if v, ok := raw["duplex"]; ok {
		var duplex string
		if err := json.Unmarshal(v, &duplex); err != nil {
			return InterfaceEthernetArgs{}, fmt.Errorf("parse duplex: %w", err)
		}
		args.Duplex = &duplex
	}

	if v, ok := raw["speed"]; ok {
		var speed string
		if err := json.Unmarshal(v, &speed); err != nil {
			return InterfaceEthernetArgs{}, fmt.Errorf("parse speed: %w", err)
		}
		args.Speed = &speed
	}

	if v, ok := raw["mac"]; ok {
		var mac string
		if err := json.Unmarshal(v, &mac); err != nil {
			return InterfaceEthernetArgs{}, fmt.Errorf("parse mac: %w", err)
		}
		args.MAC = &mac
	}

	return args, nil
}

// parseMultiValue handles VyOS fields that return a single string or an array.
func parseMultiValue(data json.RawMessage) ([]string, error) {
	// Try array first.
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}
	// Fall back to single string.
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("not a string or string array: %s", string(data))
	}
	return []string{single}, nil
}

// parseIntFromJSON handles VyOS numeric fields returned as either strings or numbers.
func parseIntFromJSON(data json.RawMessage) (int, error) {
	// Try string first (VyOS often returns numbers as strings).
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		return strconv.Atoi(s)
	}
	// Fall back to number.
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return 0, fmt.Errorf("not a string or number: %s", string(data))
	}
	return n, nil
}

// toSet converts a string slice to a set (map[string]bool).
func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
