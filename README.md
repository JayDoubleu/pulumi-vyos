# pulumi-vyos

A native Pulumi provider for managing VyOS network appliance configuration.

Unlike Terraform bridge providers, pulumi-vyos talks directly to the VyOS HTTP
API. Resources are code-generated from the VyOS XML interface definitions,
covering the full configuration surface with over 560 resource types.

## Installation

Python (local development):

```bash
pip install ./sdk/python
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

See [`examples/python/`](examples/python/) for the full runnable example.

## Available Resources

The provider generates over 560 resources from VyOS XML interface definitions,
covering interfaces, firewall, NAT, routing protocols, system settings, services,
VPN, and more. Every resource supports full CRUD operations and state refresh.

Resource names follow the VyOS config hierarchy. For example:
- `vyos.InterfaceEthernet` for `interfaces ethernet <name>`
- `vyos.FirewallIPv4NameRule` for `firewall ipv4 name <name> rule <n>`
- `vyos.ServiceDHCPServerSharedNetworkNameSubnet` for DHCP subnets

## Development

See [CLAUDE.md](CLAUDE.md) for build instructions and project structure, and
[DESIGN.md](DESIGN.md) for architecture decisions and development plan.

```bash
make build    # generate + compile + schema + SDKs
make test     # run unit tests
make lint     # run golangci-lint
```
