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

## Project Structure

```
pulumi-vyos/
  go.mod, go.sum          # Go module
  Makefile                 # Build targets: provider, schema, codegen, test, lint
  .golangci.yml            # Lint config (golangci-lint v2)
  .gitignore               # Ignores bin/, sdk/, test VM artifacts
  schema.json              # Generated Pulumi schema
  provider/
    provider.go            # Provider builder, resource registration
    config.go              # ProviderConfig (host, apiKey, port, protocol, insecure)
    resource_system_hostname.go  # First resource: system hostname CRUD
    resource_system_hostname_test.go
    resource_interface_ethernet.go  # Ethernet interface: batch ops, multi-value, valueless booleans
    resource_interface_ethernet_test.go
    cmd/pulumi-resource-vyos/
      main.go              # Provider binary entry point
    vyosclient/
      client.go            # HTTP client, API interface, mutex serialization
      configure.go         # Set/Delete/Batch operations
      retrieve.go          # ShowConfig/Exists
      configfile.go        # SaveConfig
      errors.go            # APIError, AuthError types
      *_test.go            # Unit tests with httptest.Server mocks
  sdk/                     # Generated SDKs (go, nodejs, python)
  examples/                # Example programs (yaml, go, python, typescript)
  test/
    vm/                    # VyOS QEMU VM for integration testing
      cloud-init/          # Cloud-init config (user-data, meta-data)
      run.sh               # VM lifecycle helper script
    integration/           # Integration tests (build-tagged)
  codegen/
    xml-samples/           # Sample VyOS XML interface definitions
  .github/workflows/
    ci.yml                 # CI: lint, test, build, schema verify, SDK gen
```

## Development

- Language: Go 1.25+
- Provider SDK: `github.com/pulumi/pulumi-go-provider` v1.3.0
- Target SDKs: TypeScript, Python, Go (auto-generated)
- VyOS API: HTTP REST (`/configure`, `/retrieve`, `/config-file`)

### Build

```bash
make provider     # Build provider binary
make schema       # Generate schema.json
make codegen      # Generate all SDKs
make build        # provider + codegen
make test              # Run all unit tests
make test_integration  # Run integration tests (needs VyOS VM)
make lint         # Run golangci-lint
```

### Testing

- **Unit tests**: `make test` or `go test -race ./provider/...` (uses httptest mocks)
- **Integration tests**: `make test_integration` or `go test -v -count=1 -tags=integration ./test/integration/...` (needs VyOS VM)
- **VyOS VM**: `test/vm/run.sh start` (QEMU with cloud-init, API on port 8443)
- Env vars for integration tests: `VYOS_HOST` (default: localhost), `VYOS_API_PORT` (default: 8443), `VYOS_API_KEY` (default: integration-test-key)

### Adding Resources

1. Create `provider/resource_<name>.go` with struct, args, state, CRUD methods
2. Register in `provider/provider.go` via `infer.Resource()`
3. Add unit tests in `provider/resource_<name>_test.go`
4. Run `make build` to regenerate schema and SDKs

## Git Identity

- `user.name="Jay W"`
- `user.email="git.jaydoubleu@gmail.com"`
