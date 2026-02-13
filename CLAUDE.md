# pulumi-vyos

Native Pulumi provider for VyOS network appliances.

## Project Overview

This is a **native** Pulumi provider (not a Terraform bridge) built with the
Pulumi Go Provider SDK (`github.com/pulumi/pulumi-go-provider`). It talks
directly to the VyOS HTTP API.

See `DESIGN.md` for full research, architecture decisions, and development plan.

## Key Architecture Decisions

- **Native Go provider** using the `infer` package (not Terraform bridge)
- **Code generation** from VyOS XML interface definitions (`vyos/vyos-1x`)
- **Provider-side mutex** to serialize HTTP requests (VyOS API is not concurrent-safe)
- **Medium resource granularity** -- one resource per logical config object (tag node)
- **Batch operations** where VyOS requires atomic commits (DHCP, VXLAN, tunnels, etc.)

## Development

- Language: Go
- Provider SDK: `github.com/pulumi/pulumi-go-provider`
- Target SDKs: TypeScript, Python, Go (auto-generated)
- VyOS API: HTTP REST (`/configure`, `/retrieve`, `/config-file`)

## Git Identity

- `user.name="Jay W"`
- `user.email="git.jaydoubleu@gmail.com"`
