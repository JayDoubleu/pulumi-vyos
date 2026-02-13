package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// SystemHostname manages the VyOS system hostname configuration.
type SystemHostname struct{}

// SystemHostnameArgs defines the inputs for the SystemHostname resource.
type SystemHostnameArgs struct {
	Hostname string `pulumi:"hostname"`
}

// SystemHostnameState defines the outputs (persisted state) for SystemHostname.
type SystemHostnameState struct {
	SystemHostnameArgs
}

// Annotate provides schema metadata for the resource.
func (*SystemHostname) Annotate(a infer.Annotator) {
	a.Describe(new(SystemHostname), "Manages the VyOS system hostname.")
}

// Annotate provides schema metadata for the inputs.
func (args *SystemHostnameArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Hostname, "The system hostname.")
}

// Create sets the system hostname on VyOS.
func (SystemHostname) Create(
	ctx context.Context,
	req infer.CreateRequest[SystemHostnameArgs],
) (infer.CreateResponse[SystemHostnameState], error) {
	if req.DryRun {
		return infer.CreateResponse[SystemHostnameState]{
			ID:     req.Inputs.Hostname,
			Output: SystemHostnameState{SystemHostnameArgs: req.Inputs},
		}, nil
	}

	client := getClient(ctx)
	if err := client.Set(ctx, []string{"system", "host-name"}, req.Inputs.Hostname); err != nil {
		return infer.CreateResponse[SystemHostnameState]{}, fmt.Errorf("set system host-name: %w", err)
	}

	return infer.CreateResponse[SystemHostnameState]{
		ID:     req.Inputs.Hostname,
		Output: SystemHostnameState{SystemHostnameArgs: req.Inputs},
	}, nil
}

// Read fetches the current hostname from VyOS.
func (SystemHostname) Read(
	ctx context.Context,
	_ infer.ReadRequest[SystemHostnameArgs, SystemHostnameState],
) (infer.ReadResponse[SystemHostnameArgs, SystemHostnameState], error) {
	client := getClient(ctx)
	data, err := client.ShowConfig(ctx, []string{"system"})
	if err != nil {
		return infer.ReadResponse[SystemHostnameArgs, SystemHostnameState]{},
			fmt.Errorf("read system config: %w", err)
	}

	var sysConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &sysConfig); err != nil {
		return infer.ReadResponse[SystemHostnameArgs, SystemHostnameState]{},
			fmt.Errorf("unmarshal system config: %w", err)
	}

	var hostname string
	if raw, ok := sysConfig["host-name"]; ok {
		if err := json.Unmarshal(raw, &hostname); err != nil {
			return infer.ReadResponse[SystemHostnameArgs, SystemHostnameState]{},
				fmt.Errorf("unmarshal host-name: %w", err)
		}
	}

	args := SystemHostnameArgs{Hostname: hostname}
	state := SystemHostnameState{SystemHostnameArgs: args}

	return infer.ReadResponse[SystemHostnameArgs, SystemHostnameState]{
		ID:     hostname,
		Inputs: args,
		State:  state,
	}, nil
}

// Update changes the system hostname on VyOS.
func (SystemHostname) Update(
	ctx context.Context,
	req infer.UpdateRequest[SystemHostnameArgs, SystemHostnameState],
) (infer.UpdateResponse[SystemHostnameState], error) {
	if req.DryRun {
		return infer.UpdateResponse[SystemHostnameState]{
			Output: SystemHostnameState{SystemHostnameArgs: req.Inputs},
		}, nil
	}

	client := getClient(ctx)
	if err := client.Set(ctx, []string{"system", "host-name"}, req.Inputs.Hostname); err != nil {
		return infer.UpdateResponse[SystemHostnameState]{}, fmt.Errorf("update system host-name: %w", err)
	}

	return infer.UpdateResponse[SystemHostnameState]{
		Output: SystemHostnameState{SystemHostnameArgs: req.Inputs},
	}, nil
}

// Delete removes the system hostname from VyOS configuration.
func (SystemHostname) Delete(
	ctx context.Context,
	_ infer.DeleteRequest[SystemHostnameState],
) (infer.DeleteResponse, error) {
	client := getClient(ctx)
	if err := client.Delete(ctx, []string{"system", "host-name"}); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("delete system host-name: %w", err)
	}
	return infer.DeleteResponse{}, nil
}

// getClient retrieves the VyOS API client from the provider config context.
func getClient(ctx context.Context) vyosclient.API {
	return infer.GetConfig[Config](ctx).Client()
}
