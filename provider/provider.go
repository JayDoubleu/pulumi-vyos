// Package provider implements the VyOS Pulumi provider.
package provider

import (
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Version is set at build time via ldflags.
var Version string

// Name is the Pulumi package name for this provider.
const Name = "vyos"

// Provider builds and returns the VyOS Pulumi provider.
func Provider() p.Provider {
	prov, err := infer.NewProviderBuilder().
		WithDisplayName("VyOS").
		WithDescription("A Pulumi provider for managing VyOS network appliances.").
		WithGoImportPath("github.com/jaydoubleu/pulumi-vyos/sdk/go/vyos").
		WithConfig(infer.Config(&Config{})).
		WithResources(append(GeneratedResources(), infer.Resource(ConfigFileSave{}))...).
		WithComponents(
			infer.ComponentF(NewStaticRouteComplete),
			infer.ComponentF(NewFirewallIPv4Ruleset),
		).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		}).
		Build()
	if err != nil {
		panic(fmt.Errorf("unable to build provider: %w", err))
	}
	return prov
}
