// Package main is the entry point for the VyOS Pulumi provider plugin.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jaydoubleu/pulumi-vyos/provider"
)

func main() {
	err := provider.Provider().Run(context.Background(), provider.Name, provider.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
