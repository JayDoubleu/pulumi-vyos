# pulumi-vyos Design Document

Native Pulumi provider for VyOS network appliances, built with the Pulumi Go Provider SDK.

## Research Summary (February 2026)

### VyOS API Landscape

#### HTTP/REST API

VyOS exposes an HTTP API starting from v1.3 (Equuleus). Architecture:

- **Nginx** reverse proxy for SSL termination
- **FastAPI-based `vyos-http-api-server`** handling requests
- **Unix socket** between Nginx and API server (`/run/api.sock`)

Authentication: API key included in request payload, configured via
`set service https api keys id <id> key <key>`.

All endpoints are POST (except `/info` which is GET):

| Endpoint       | Purpose                                     |
|----------------|---------------------------------------------|
| `/configure`   | Apply config changes (set/delete operations) |
| `/retrieve`    | Get configuration data                       |
| `/show`        | Run operational show commands                |
| `/generate`    | Run generate commands (e.g., crypto keys)    |
| `/config-file` | Save/load configuration files                |
| `/image`       | System image management                      |
| `/reset`       | Reset operations                             |
| `/reboot`      | Reboot router                                |
| `/poweroff`    | Power off router                             |
| `/info`        | Get system information (GET, public)         |

All responses follow:
```json
{
  "success": true,
  "data": "...",
  "error": null
}
```

#### GraphQL API

Available at `/graphql`, supports both API key and JWT authentication.
Schema is statically generated during package build.

#### NETCONF

**Not supported.** There is a dev tracker item (T68) for future support but
nothing exists today. Ansible's `vyos.vyos` collection works entirely over SSH.

### CRUD Mapping to VyOS API

#### Create / Update

```bash
curl -k -X POST 'https://vyos/configure' \
  --form data='{"op": "set", "path": ["interfaces", "ethernet", "eth0", "address"], "value": "192.168.1.1/24"}' \
  --form key='MY-API-KEY'
```

- `op: "set"` for both create and update
- `path` is an array of config tree segments
- `value` is the leaf value to set
- Auto-commits on success (no separate commit step)

#### Read

```bash
# Get config subtree
curl -k -X POST 'https://vyos/retrieve' \
  --form data='{"op": "showConfig", "path": ["interfaces", "ethernet", "eth0"]}' \
  --form key='MY-API-KEY'

# Get multi-valued node as array
curl -k -X POST 'https://vyos/retrieve' \
  --form data='{"op": "returnValues", "path": ["interfaces", "ethernet"]}' \
  --form key='MY-API-KEY'

# Boolean existence check
curl -k -X POST 'https://vyos/retrieve' \
  --form data='{"op": "exists", "path": ["interfaces", "ethernet", "eth0"]}' \
  --form key='MY-API-KEY'
```

Empty path `[]` retrieves the entire configuration.

#### Delete

```bash
curl -k -X POST 'https://vyos/configure' \
  --form data='{"op": "delete", "path": ["interfaces", "dummy", "dum1"]}' \
  --form key='MY-API-KEY'
```

Also auto-commits.

#### Batching (Atomic Multi-Operation Commits)

Pass an array of operations for a single atomic commit:

```bash
curl -k -X POST 'https://vyos/configure' \
  --form data='[
    {"op": "set", "path": ["interfaces", "vxlan", "vxlan1", "remote", "203.0.113.99"]},
    {"op": "set", "path": ["interfaces", "vxlan", "vxlan1", "vni", "1"]},
    {"op": "delete", "path": ["interfaces", "dummy", "dum1"]}
  ]' \
  --form key='MY-API-KEY'
```

Some VyOS components (DHCP, PPPoE, IPSec, VXLAN, tunnels) require full
configuration in a single commit for validation to pass.

#### Commit-Confirm (Rollback Safety)

Add `confirm_time` (integer, minutes) for auto-rollback if not confirmed:

```bash
curl -k -X POST 'https://vyos/configure' \
  --form data='{"op": "set", "path": ["system", "host-name"], "value": "test", "confirm_time": 1}' \
  --form key='MY-API-KEY'
```

Confirm with: `{"op": "confirm"}`

#### Idempotency

Setting the same value twice is a no-op. VyOS treats `set` as idempotent
for identical values. Formatting matters though -- abbreviated commands may
not behave idempotently.

### Critical Constraint: Concurrency

**VyOS's HTTP API cannot handle concurrent requests safely.**

- Global session lock with single-threaded config handling (Flask + uWSGI)
- Concurrent requests cause 504 Gateway Timeouts, segfaults, config corruption
- All existing Terraform providers suffer from this
- Users must use `terraform apply -parallelism=1`

**Mitigation for this provider:** Implement a provider-side mutex so only one
HTTP request hits VyOS at a time, regardless of Pulumi's parallelism setting.
This is invisible to users and the cleanest solution.

### VyOS Configuration Model

Tree-based, similar to JunOS, with three states:

1. **Running configuration** -- currently active
2. **Working configuration** -- modified in config mode, not yet committed
3. **Saved configuration** -- persisted to disk via `save`

#### Node Types

| Type       | Description                                    | Example                          |
|------------|------------------------------------------------|----------------------------------|
| `node`     | Intermediate, holds children only              | `system`, `interfaces`           |
| `tagNode`  | Named container for dynamic instances          | `ethernet eth0`, `name MyPolicy` |
| `leafNode` | Terminal, holds a value                         | `address`, `description`         |

Leaf node value types:
- **Single-value**: one string/number (e.g., `host-name`)
- **Multi-value**: list of values (e.g., `name-server`)
- **Valueless**: boolean flag, presence means true (e.g., `disable-forwarding`)

#### Set-Based CLI

```
set interfaces ethernet eth0 address 192.168.1.1/24
set system host-name router01
delete interfaces ethernet eth0 address 192.168.1.1/24
commit
save
```

### VyOS XML Interface Definitions (Schema Source)

The `vyos/vyos-1x` repo contains **125 XML definition files** in
`interface-definitions/*.xml.in` that fully describe every configuration node.

These are the authoritative schema for code generation.

#### Example: Static ARP

```xml
<node name="protocols">
  <children>
    <node name="static">
      <children>
        <node name="arp" owner="${vyos_conf_scripts_dir}/protocols_static_arp.py">
          <properties>
            <help>Static ARP translation</help>
            <priority>481</priority>
          </properties>
          <children>
            <tagNode name="interface">
              <properties>
                <help>Interface configuration</help>
                <constraint>
                  #include <include/constraint/interface-name.xml.i>
                </constraint>
              </properties>
              <children>
                <tagNode name="address">
                  <properties>
                    <help>IP address for static ARP entry</help>
                    <valueHelp>
                      <format>ipv4</format>
                      <description>IPv4 destination address</description>
                    </valueHelp>
                    <constraint>
                      <validator name="ipv4-address"/>
                    </constraint>
                  </properties>
                  <children>
                    #include <include/generic-description.xml.i>
                    #include <include/interface/mac.xml.i>
                  </children>
                </tagNode>
              </children>
            </tagNode>
          </children>
        </node>
      </children>
    </node>
  </children>
</node>
```

#### Constraint System

- **Regex**: `<regex>[a-zA-Z0-9][\w\-\.]*</regex>`
- **Validators**: `<validator name="ipv4-address"/>` (60+ executables in `src/validators/`)
- **Constraint groups**: logical AND of multiple constraints
- **Error messages**: `<constraintErrorMessage>...</constraintErrorMessage>`

#### Property Markers

- `<valueless/>` -- boolean flag (no value)
- `<multi/>` -- list (multiple values allowed)
- `<hidden/>` -- hidden from unprivileged users
- `<secret/>` -- value masked in output
- `<defaultValue>` -- default if not set
- `<priority>` -- processing order (integer)

#### Include System

Reusable fragments via C preprocessor includes:

```xml
#include <include/generic-description.xml.i>
#include <include/constraint/alpha-numeric-hyphen-underscore-dot.xml.i>
```

These must be resolved (preprocessed) before XML parsing.

#### Type Inference from XML

Logic proven by thomasfinstad's Terraform provider:

- `<valueless/>` present --> `bool`
- All `<valueHelp>` formats are `u32` --> `number`
- `<multi/>` present --> `[]string` (or `[]number`)
- Everything else --> `string`

### Existing Automation Ecosystem

#### Ansible `vyos.vyos` Collection

Official, SSH-based (`network_cli`), comprehensive modules covering firewall,
BGP, interfaces, static routes, hostname, users, etc. **Not useful as a code
foundation** (SSH/CLI text parsing), but the resource model is a good reference
for what users expect.

#### Terraform Providers

| Provider                            | Status                | Approach                          |
|-------------------------------------|-----------------------|-----------------------------------|
| `Foltik/vyos` (v0.3.4, May 2025)   | Active, limited       | Generic `vyos_config` resources   |
| `thomasfinstad/vyos-rolling`        | **Archived** Mar 2025 | Auto-generated from VyOS XML      |
| `TGNThump/vyos`                     | Dormant since 2023    | Basic                             |

The thomasfinstad provider is the most relevant -- it proved XML-based code
generation works and produced comprehensive resource coverage. The code
generation pipeline (Go) can be adapted for Pulumi.

#### Python: pyvyos

Official Python SDK for VyOS HTTP API (`pip install pyvyos`). Thin wrapper
around the REST endpoints. Not directly useful for the Go provider but good
reference for API behavior.

### Pulumi Provider Development

#### Chosen Approach: Native Go Provider SDK

Using `github.com/pulumi/pulumi-go-provider` with the `infer` package.

Why native over Terraform bridge:
- No dependency on a Terraform provider (the good ones are archived/dormant)
- Clean modeling of VyOS's commit semantics
- Provider-side mutex for concurrency (bridge can't do this)
- First-class Pulumi experience
- Code-first with auto-generated schema

#### Resource Lifecycle (infer package)

Only `Create` is required. Optional methods with sensible defaults:

| Method   | Purpose                          | Default if not implemented         |
|----------|----------------------------------|------------------------------------|
| `Create` | Create resource (REQUIRED)       | --                                 |
| `Read`   | Refresh state from device        | Validates serialization            |
| `Update` | In-place mutation                | Replace (delete + create)          |
| `Delete` | Remove resource                  | No-op                              |
| `Diff`   | Detect changes                   | Structural field comparison        |
| `Check`  | Validate inputs                  | Confirms struct serialization      |

#### Provider Config

```go
type ProviderConfig struct {
    Host     string `pulumi:"host"`
    APIKey   string `pulumi:"apiKey" provider:"secret"`
    Port     *int   `pulumi:"port,optional"`
    Protocol *string `pulumi:"protocol,optional"`
    Insecure *bool  `pulumi:"insecure,optional"`
}
```

Accessed in resource methods via `infer.GetConfig[ProviderConfig](ctx)`.

#### Resource Example

```go
type InterfaceEthernet struct{}

type InterfaceEthernetArgs struct {
    Name        string   `pulumi:"name"`
    Address     []string `pulumi:"address,optional"`
    Description *string  `pulumi:"description,optional"`
    Duplex      *string  `pulumi:"duplex,optional"`
    Speed       *string  `pulumi:"speed,optional"`
    MTU         *int     `pulumi:"mtu,optional"`
}

type InterfaceEthernetState struct {
    InterfaceEthernetArgs
    MAC string `pulumi:"mac"`
}

func (*InterfaceEthernet) Create(ctx context.Context, req infer.CreateRequest[InterfaceEthernetArgs]) (
    infer.CreateResponse[InterfaceEthernetState], error) {
    config := infer.GetConfig[ProviderConfig](ctx)
    client := vyos.NewClient(config.Host, config.APIKey, *config.Port)

    ops := []vyos.Operation{
        {Op: "set", Path: []string{"interfaces", "ethernet", req.Inputs.Name, "address"}, Value: req.Inputs.Address[0]},
    }
    if req.Inputs.Description != nil {
        ops = append(ops, vyos.Operation{
            Op: "set", Path: []string{"interfaces", "ethernet", req.Inputs.Name, "description"},
            Value: *req.Inputs.Description,
        })
    }
    // ... more fields

    if err := client.ConfigureBatch(ops); err != nil {
        return infer.CreateResponse[InterfaceEthernetState]{}, err
    }

    return infer.CreateResponse[InterfaceEthernetState]{
        ID: req.Inputs.Name,
        Output: InterfaceEthernetState{
            InterfaceEthernetArgs: req.Inputs,
            MAC:                   "read-from-device",
        },
    }, nil
}
```

### Resource Granularity Decision

**Chosen: Medium granularity** -- one resource per logical object (tag node).

Map VyOS tag nodes to Pulumi resources. This matches how users think about
config and aligns with the Ansible module design:

- `vyos.InterfaceEthernet` -- manages `interfaces ethernet <name>` subtree
- `vyos.FirewallRule` -- manages a single firewall rule
- `vyos.StaticRoute` -- manages `protocols static route <prefix>` subtree
- `vyos.NatSourceRule` -- manages a NAT source rule
- `vyos.BgpNeighbor` -- manages a BGP neighbor

The XML `owner` attribute naturally groups config into these logical objects
(it points to the Python script that processes that subtree).

Component resources can later compose these into higher-level abstractions
(e.g., a full firewall policy with multiple rules).

## Development Plan

### Phase 1: Foundation (1-2 weeks)

- [ ] Scaffold provider with `pulumi-provider-boilerplate`
- [ ] Implement `ProviderConfig` (host, API key, port, TLS)
- [ ] Build VyOS HTTP API client in Go
  - Thin wrapper around `/configure`, `/retrieve`, `/config-file`
  - Mutex for request serialization (concurrency safety)
  - Retry logic with backoff
  - Batch operation support
- [ ] Hand-craft one resource: `vyos.InterfaceEthernet` with full CRUD
- [ ] Set up testing against a real VyOS instance (VM or container)
- [ ] Basic CI (build, lint, unit tests)

### Phase 2: Code Generator (2-3 weeks)

- [ ] Port/adapt XML parsing from `thomasfinstad/terraform-provider-vyos-rolling`
  - XML unmarshaling to Go structs
  - Include file preprocessing
  - Type inference (valueless->bool, u32->number, multi->list, etc.)
- [ ] Build Go templates that emit Pulumi resource code
  - Struct definitions with `pulumi:` tags
  - CRUD method implementations
  - Annotate methods (descriptions, defaults)
- [ ] Generate resources for core config areas:
  - Interfaces (ethernet, bonding, bridge, vxlan, wireguard, etc.)
  - Firewall (rules, groups, zones)
  - NAT (source, destination rules)
  - Routing (static, BGP, OSPF)
  - System (DNS, NTP, syslog, users)
  - Services (DHCP, SSH, HTTPS)
- [ ] Validate generated code compiles and passes basic tests

### Phase 3: Polish (1-2 weeks)

- [ ] Implement `Read` for all resources (state refresh / drift detection)
- [ ] Add `Check` with validation from XML constraints
- [ ] `config-file save` as provider-level option or explicit resource
- [ ] Integration test suite
- [ ] Documentation and usage examples
- [ ] Multi-language SDK generation and testing (TS, Python, Go at minimum)

### Phase 4: Full Coverage and Release (ongoing)

- [ ] Generate all 125 XML definitions
- [ ] Component resources for common patterns
- [ ] Publish to Pulumi Registry
- [ ] CI pipeline for regeneration when VyOS updates XML definitions
- [ ] Community feedback and iteration

## Key References

### Repositories

- VyOS core: `github.com/vyos/vyos-1x` (XML definitions, Python config system)
- Archived TF provider: `github.com/thomasfinstad/terraform-provider-vyos-rolling` (code gen reference)
- Active TF provider: `github.com/foltik/terraform-provider-vyos` (API client reference)
- pyvyos: `github.com/vyos-contrib/pyvyos` (Python API client reference)
- Pulumi Go Provider SDK: `github.com/pulumi/pulumi-go-provider`
- Pulumi provider boilerplate: `github.com/pulumi/pulumi-provider-boilerplate`

### Documentation

- VyOS HTTP API: https://docs.vyos.io/en/latest/automation/vyos-api.html
- VyOS CLI/config model: https://docs.vyos.io/en/latest/cli.html
- Pulumi Go Provider SDK: https://www.pulumi.com/docs/iac/guides/building-extending/providers/sdks/pulumi-go-provider-sdk/
- Pulumi provider architecture: https://www.pulumi.com/docs/iac/guides/building-extending/providers/provider-architecture/
- Ansible vyos.vyos collection: https://docs.ansible.com/ansible/latest/collections/vyos/vyos/index.html

### Similar Pulumi Providers (for reference)

- `pulumi-unifi` -- bridged, manages UniFi network devices
- `pulumi-fortios` -- bridged, manages FortiGate firewalls
- Cisco `iosxe`, `nxos`, `aci` -- bridged, network device management
- `pulumi-junipermist` -- Juniper Mist
