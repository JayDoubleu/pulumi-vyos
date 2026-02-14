# Changelog

## 0.1.0-alpha.1

Initial alpha release.

### Features

- **600 resources** generated from VyOS XML interface definitions (interfaces,
  firewall, NAT, routing, system, services, VPN, and more)
- **3 SDK languages**: Python, TypeScript, Go
- **Code generation pipeline** from VyOS XML (`vyos/vyos-1x`) to typed Pulumi
  resources with full CRUD support
- **Component resources**: `StaticRouteComplete` (route + next-hops) and
  `FirewallIPv4Ruleset` (firewall policy + rules)
- **Config file save**: provider-level `saveConfig` flag and explicit
  `ConfigFileSave` resource
- **Client-side validation**: `Check` methods with regex and numeric range
  constraints from VyOS XML definitions
- **Provider-side mutex** for safe serialization of VyOS API calls
