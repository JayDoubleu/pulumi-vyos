# Contributing to pulumi-vyos

Thank you for your interest in contributing to pulumi-vyos!

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/jaydoubleu/pulumi-vyos.git
   cd pulumi-vyos
   ```

2. Initialize the VyOS XML submodule:
   ```bash
   git submodule update --init codegen/vyos-1x
   ```

3. Build the provider and generate SDKs:
   ```bash
   make build
   ```

## Building and Testing

```bash
make build        # Full pipeline: generate + compile + schema + SDKs
make test         # Run all unit tests
make lint         # Run golangci-lint
make generate     # Regenerate resource files from VyOS XML
make provider     # Build provider binary only
make clean        # Remove generated files and build artifacts
```

## Generated Code

Files matching `provider/resource_gen_*.go` are auto-generated from VyOS XML
interface definitions. Do not edit these files directly. Instead, modify the
templates in `codegen/generate/templates/` and run `make generate`.

## Submitting Changes

1. Fork the repository and create a feature branch
2. Make your changes
3. Run `make test && make lint` to verify
4. Submit a pull request

## Architecture

See [DESIGN.md](DESIGN.md) for architecture decisions and the development plan.
