package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/provider/vyosclient"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// mockClient implements vyosclient.API for testing.
type mockClient struct {
	setFn            func(ctx context.Context, path []string, value any) error
	deleteFn         func(ctx context.Context, path []string) error
	batchConfigureFn func(ctx context.Context, ops []vyosclient.Operation) error
	showConfigFn     func(ctx context.Context, path []string) (json.RawMessage, error)
	existsFn         func(ctx context.Context, path []string) (bool, error)
	saveConfigFn     func(ctx context.Context) error
}

func (m *mockClient) Set(ctx context.Context, path []string, value any) error {
	if m.setFn != nil {
		return m.setFn(ctx, path, value)
	}
	return nil
}

func (m *mockClient) Delete(ctx context.Context, path []string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, path)
	}
	return nil
}

func (m *mockClient) BatchConfigure(ctx context.Context, ops []vyosclient.Operation) error {
	if m.batchConfigureFn != nil {
		return m.batchConfigureFn(ctx, ops)
	}
	return nil
}

func (m *mockClient) ShowConfig(ctx context.Context, path []string) (json.RawMessage, error) {
	if m.showConfigFn != nil {
		return m.showConfigFn(ctx, path)
	}
	return nil, nil
}

func (m *mockClient) Exists(ctx context.Context, path []string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, path)
	}
	return false, nil
}

func (m *mockClient) SaveConfig(ctx context.Context) error {
	if m.saveConfigFn != nil {
		return m.saveConfigFn(ctx)
	}
	return nil
}

var _ vyosclient.API = (*mockClient)(nil)

func TestSystemHostname_CreateDryRun(t *testing.T) {
	t.Parallel()
	resource := SystemHostname{}

	resp, err := resource.Create(context.Background(), infer.CreateRequest[SystemHostnameArgs]{
		Name:   "test-hostname",
		Inputs: SystemHostnameArgs{Hostname: "router1"},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Create dry run error: %v", err)
	}
	if resp.ID != "router1" {
		t.Errorf("ID = %q, want %q", resp.ID, "router1")
	}
	if resp.Output.Hostname != "router1" {
		t.Errorf("Hostname = %q, want %q", resp.Output.Hostname, "router1")
	}
}

func TestSystemHostname_UpdateDryRun(t *testing.T) {
	t.Parallel()
	resource := SystemHostname{}

	resp, err := resource.Update(context.Background(), infer.UpdateRequest[SystemHostnameArgs, SystemHostnameState]{
		ID:     "router1",
		State:  SystemHostnameState{SystemHostnameArgs: SystemHostnameArgs{Hostname: "router1"}},
		Inputs: SystemHostnameArgs{Hostname: "router2"},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Update dry run error: %v", err)
	}
	if resp.Output.Hostname != "router2" {
		t.Errorf("Hostname = %q, want %q", resp.Output.Hostname, "router2")
	}
}
