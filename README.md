# pulumi-vyos

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

> **Alpha**: This provider is under active development. APIs may change
> between releases. Please report issues on GitHub.

A native Pulumi provider for managing VyOS network appliance configuration.

Unlike Terraform bridge providers, pulumi-vyos talks directly to the VyOS HTTP
API. Resources are code-generated from the VyOS XML interface definitions.
With the current `codegen/vyos-1x` submodule snapshot, `make generate` emits
600 resource types after deduplication.

## Installation

SDKs are not yet published to package registries. Build from source:

```bash
git clone https://github.com/jaydoubleu/pulumi-vyos.git
cd pulumi-vyos
git submodule update --init
make build
```

This generates the provider binary and SDKs for Python, TypeScript, and Go
in the `sdk/` directory. To use a generated SDK in your project, install it
from the local path:

### Python

```bash
pip install ./sdk/python
```

### TypeScript / JavaScript

```bash
npm install ./sdk/nodejs
```

### Go

```go
import "github.com/jaydoubleu/pulumi-vyos/sdk/go/vyos"
```

## Configuration

Configure the provider via `pulumi config`:

```bash
pulumi config set vyos:host 192.168.1.1
pulumi config set vyos:apiKey --secret MY_API_KEY
```

| Property     | Description                                      | Default |
|--------------|--------------------------------------------------|---------|
| `host`       | VyOS host address (IP or hostname)               |         |
| `apiKey`     | VyOS HTTP API key                                |         |
| `port`       | API port                                         | `443`   |
| `protocol`   | Protocol (`https` or `http`)                     | `https` |
| `insecure`   | Skip TLS certificate verification                | `false` |
| `saveConfig` | Auto-save running config to disk after every op  | `false` |

## Python Example

```python
import pulumi
import pulumi_vyos as vyos

# System hostname
hostname = vyos.SystemHostName("hostname", host_name="my-vyos-router")

# Dummy interface
dummy = vyos.InterfaceDummy("dum0",
    name="dum0",
    description="Management network",
    address=["10.0.0.1/24"],
    mtu=1500,
)

# Firewall address group
web_servers = vyos.FirewallGroupAddressGroup("web-servers",
    name="web-servers",
    description="Web server pool",
    address=["10.0.1.10", "10.0.1.11", "10.0.1.12"],
)

# NTP server
ntp = vyos.ServiceNTPServer("pool-ntp",
    name="pool.ntp.org",
    pool=True,
    prefer=True,
)

# Static route with next-hop (next-hop depends on the parent route)
route = vyos.ProtocolStaticRoute("mgmt-route",
    name="10.10.0.0/16",
    description="Management network route",
)
next_hop = vyos.ProtocolStaticRouteNextHop("mgmt-gw",
    route_name="10.10.0.0/16",
    name="10.0.0.254",
    opts=pulumi.ResourceOptions(depends_on=[route]),
)

# NAT masquerade rule
nat_rule = vyos.NATSourceRule("masquerade",
    name="100",
    outbound_interface_name="eth0",
    translation_address="masquerade",
)

# Save config to disk
save = vyos.ConfigFileSave("save-config",
    triggers={"version": "1"},
)

pulumi.export("hostname", hostname.host_name)
```

See [`examples/`](examples/) for runnable examples in Python, TypeScript, Go, and YAML.

## Available Resources

The provider currently generates 600 resources from VyOS XML interface
definitions (123 XML files parsed, 1 skipped in the current snapshot). This
covers interfaces, firewall, NAT, routing protocols, system settings, services,
VPN, and more. Every generated resource implements Create, Read, Update, and
Delete methods.

Resource names follow the VyOS config hierarchy. For example:
- `vyos.InterfaceEthernet` for `interfaces ethernet <name>`
- `vyos.FirewallIPv4NameRule` for `firewall ipv4 name <name> rule <n>`
- `vyos.ServiceDHCPServerSharedNetworkNameSubnet` for DHCP subnets

## Status

### What works

- Full CRUD for all 600 generated resources
- Client-side validation (regex and numeric range constraints)
- Component resources: `StaticRouteComplete`, `FirewallIPv4Ruleset`
- Config file save (automatic and explicit)
- Python, TypeScript, and Go SDKs

### Known limitations

- Named validators (ipv4-address, mac-address, etc.) are not yet implemented
- Import workflows are not yet documented or covered by integration tests
- VyOS API requires serial access (provider uses a mutex)
- Only tested against VyOS rolling builds

## Development

See [DESIGN.md](DESIGN.md) for architecture decisions and the development plan.

```bash
make build    # generate + compile + schema + SDKs
make test     # run unit tests
make lint     # run golangci-lint
make clean    # remove generated files and build artifacts
```

## License

Apache 2.0. See [LICENSE](LICENSE).
