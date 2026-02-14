package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ConfigFileSave persists the VyOS running configuration to disk.
// Use this resource for explicit control over when config is saved.
type ConfigFileSave struct{}

// ConfigFileSaveArgs defines the inputs for the ConfigFileSave resource.
type ConfigFileSaveArgs struct {
	Triggers map[string]string `pulumi:"triggers,optional"`
}

// ConfigFileSaveState defines the outputs for the ConfigFileSave resource.
type ConfigFileSaveState struct {
	ConfigFileSaveArgs
	SavedAt string `pulumi:"savedAt"`
}

// Annotate provides schema metadata for the resource.
func (*ConfigFileSave) Annotate(a infer.Annotator) {
	a.Describe(new(ConfigFileSave), "Saves the VyOS running configuration to disk. "+
		"Changes to the triggers map cause the config to be re-saved.")
}

// Annotate provides schema metadata for the inputs.
func (args *ConfigFileSaveArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Triggers, "Arbitrary key-value map. A change in any value triggers a re-save.")
}

// Create saves the running config to disk.
func (ConfigFileSave) Create(
	ctx context.Context,
	req infer.CreateRequest[ConfigFileSaveArgs],
) (infer.CreateResponse[ConfigFileSaveState], error) {
	now := time.Now().UTC().Format(time.RFC3339)

	if req.DryRun {
		return infer.CreateResponse[ConfigFileSaveState]{
			ID: makeConfigSaveID(req.Inputs.Triggers, now),
			Output: ConfigFileSaveState{
				ConfigFileSaveArgs: req.Inputs,
				SavedAt:            now,
			},
		}, nil
	}

	client := getClient(ctx)
	if err := client.SaveConfig(ctx); err != nil {
		return infer.CreateResponse[ConfigFileSaveState]{},
			fmt.Errorf("save config: %w", err)
	}

	return infer.CreateResponse[ConfigFileSaveState]{
		ID: makeConfigSaveID(req.Inputs.Triggers, now),
		Output: ConfigFileSaveState{
			ConfigFileSaveArgs: req.Inputs,
			SavedAt:            now,
		},
	}, nil
}

// Read returns the current state unchanged.
func (ConfigFileSave) Read(
	ctx context.Context,
	req infer.ReadRequest[ConfigFileSaveArgs, ConfigFileSaveState],
) (infer.ReadResponse[ConfigFileSaveArgs, ConfigFileSaveState], error) {
	return infer.ReadResponse[ConfigFileSaveArgs, ConfigFileSaveState]{
		ID:     req.ID,
		Inputs: req.State.ConfigFileSaveArgs,
		State:  req.State,
	}, nil
}

// Update re-saves the config when triggers change.
func (ConfigFileSave) Update(
	ctx context.Context,
	req infer.UpdateRequest[ConfigFileSaveArgs, ConfigFileSaveState],
) (infer.UpdateResponse[ConfigFileSaveState], error) {
	now := time.Now().UTC().Format(time.RFC3339)

	if req.DryRun {
		return infer.UpdateResponse[ConfigFileSaveState]{
			Output: ConfigFileSaveState{
				ConfigFileSaveArgs: req.Inputs,
				SavedAt:            now,
			},
		}, nil
	}

	client := getClient(ctx)
	if err := client.SaveConfig(ctx); err != nil {
		return infer.UpdateResponse[ConfigFileSaveState]{},
			fmt.Errorf("save config: %w", err)
	}

	return infer.UpdateResponse[ConfigFileSaveState]{
		Output: ConfigFileSaveState{
			ConfigFileSaveArgs: req.Inputs,
			SavedAt:            now,
		},
	}, nil
}

// Delete is a no-op. A saved config cannot be "un-saved".
func (ConfigFileSave) Delete(
	_ context.Context,
	_ infer.DeleteRequest[ConfigFileSaveState],
) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// makeConfigSaveID produces a deterministic ID from sorted trigger keys and a timestamp.
func makeConfigSaveID(triggers map[string]string, timestamp string) string {
	keys := make([]string, 0, len(triggers))
	for k := range triggers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, triggers[k])
	}
	b.WriteString(timestamp)

	h := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("%x", h[:16])
}
