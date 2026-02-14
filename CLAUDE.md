# pulumi-vyos

Native Pulumi provider for VyOS network appliances.

## Project Overview

This is a **native** Pulumi provider (not a Terraform bridge) built with the
Pulumi Go Provider SDK (`github.com/pulumi/pulumi-go-provider`). It talks
directly to the VyOS HTTP API.

Resources are **code-generated** from VyOS XML interface definitions (from the
`vyos/vyos-1x` repository, added as a git submodule). The generator parses all
~125 XML files and emits ~612 Go resource files covering the full VyOS config
surface.

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
  Makefile                 # Build targets: generate, provider, schema, codegen, test, lint
  .golangci.yml            # Lint config (golangci-lint v2)
  .gitignore               # Ignores bin/, sdk/, resource_gen_*.go, test VM artifacts
  schema.json              # Generated Pulumi schema
  provider/
    provider.go            # Provider builder, uses GeneratedResources()
    config.go              # ProviderConfig (host, apiKey, port, protocol, insecure, saveConfig) + getClient + saveIfEnabled
    resource_config_file_save.go # Hand-written ConfigFileSave resource (explicit save-to-disk)
    component_static_route.go          # StaticRouteComplete component (route + next-hops)
    component_firewall_ipv4_ruleset.go # FirewallIPv4Ruleset component (firewall + rules)
    resource_gen_*.go      # Generated resource files (do not edit)
    resource_gen_helpers.go      # Shared diff/parse helpers
    resource_gen_registration.go # GeneratedResources() function
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
    vyos-1x/               # Git submodule: vyos/vyos-1x XML definitions
    xml-samples/           # Sample VyOS XML interface definitions (for reference)
    cmd/generate/
      main.go              # CLI: --xml-dir, --output-dir flags
    xmlparse/
      types.go             # XML schema structs (incl. Constraint, Validator)
      preprocess.go        # Recursive #include resolution
      unmarshal.go          # Preprocess + XML unmarshal entry point
      *_test.go            # Tests with testdata samples
    model/
      ir.go                # IR types: Resource, Field, ResourceKind, FieldType, FieldConstraint
      builder.go           # XML tree -> []Resource (boundary detection, flattening)
      constraint.go        # BuildConstraint + ParseNumericRange (XML constraint extraction)
      naming.go            # Kebab-to-PascalCase, PascalCase-to-camelCase, reserved names
      typing.go            # Type inference from XML properties
      *_test.go            # Comprehensive test coverage
    generate/
      generator.go         # Template loading, execution, gofmt, file writing
      generator_test.go    # Tests for tag/leaf/nested resource generation
      templates/
        resource.go.tmpl   # Per-resource file (structs + CRUD + helpers)
        helpers.go.tmpl    # Shared helpers (generated once)
        registration.go.tmpl # GeneratedResources() function
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
make generate     # Generate resource files from VyOS XML definitions
make provider     # Build provider binary
make schema       # Generate schema.json
make codegen      # Generate all SDKs
make build        # generate + provider + codegen (full pipeline)
make test         # Run all unit tests
make test_integration  # Run integration tests (needs VyOS VM)
make lint         # Run golangci-lint
```

### Code Generation Pipeline

1. `make generate` parses VyOS XML files from `codegen/vyos-1x/interface-definitions/`
2. Resolves `#include` directives, unmarshals XML into typed structs
3. Walks the XML tree to detect resource boundaries (tagNodes, owned leafNodes)
4. Emits `provider/resource_gen_*.go` files with full CRUD implementations
5. `make build` then compiles, extracts schema, and generates SDKs

### Resource Types

- **TagNodeResource**: Named config objects (e.g., `interfaces ethernet eth0`). Uses BatchConfigure for atomic operations. Has a `Name` tag field.
- **LeafNodeResource**: Single-value config entries (e.g., `system host-name`). Uses Set/Delete directly.

### Component Resources

Hand-written components using `infer.ComponentF()` that bundle related resources:

- **StaticRouteComplete**: Route prefix + N next-hops (`component_static_route.go`)
- **FirewallIPv4Ruleset**: Firewall policy + N rules (`component_firewall_ipv4_ruleset.go`)

Components use `ctx.RegisterResource()` with type tokens (e.g. `vyos:index:ProtocolStaticRoute`)
to create child custom resources. A `childResource` type embeds `pulumi.CustomResourceState`
for these children. Args use plain Go types; outputs use `pulumi.Output` types.

### Check Validation

Resources with XML `<constraint>` elements get a `Check` method for client-side
validation before API calls. Supports regex patterns and numeric ranges. Perl-only
regex syntax (lookahead/lookbehind) is skipped at codegen time. Named validators
(ipv4-address, mac-address, etc.) are not yet covered.

### Field Types

- **StringField** (`*string`): Standard string value
- **IntField** (`*int`): Integer value (VyOS `u32` types)
- **BoolField** (`*bool`): Valueless flag (set path only, no value)
- **MultiField** (`[]string`): Multi-value field (each value appended to path)

### Testing

- **Unit tests**: `make test` or `go test -race ./codegen/... ./provider/...`
- **Integration tests**: `make test_integration` (needs VyOS VM)
- **VyOS VM**: `test/vm/run.sh start` (QEMU with cloud-init, API on port 8443)
- Env vars for integration tests: `VYOS_HOST` (default: localhost), `VYOS_API_PORT` (default: 8443), `VYOS_API_KEY` (default: integration-test-key)

## Git Identity

- `user.name="Jay W"`
- `user.email="git.jaydoubleu@gmail.com"`
