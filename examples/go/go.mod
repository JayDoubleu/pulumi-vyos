module github.com/jaydoubleu/pulumi-vyos/examples/go

go 1.25.5

// Use locally generated SDK: run "make build" from the repo root first.
replace github.com/jaydoubleu/pulumi-vyos/sdk/go/vyos => ../../sdk/go/vyos

require (
	github.com/jaydoubleu/pulumi-vyos/sdk/go/vyos v0.0.0
	github.com/pulumi/pulumi/sdk/v3 v3.217.0
)
