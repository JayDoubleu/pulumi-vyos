import * as vyos from "@jaydoubleu/vyos";

// System hostname
const hostname = new vyos.SystemHostName("hostname", {
    hostName: "my-vyos-router",
});

// Dummy interface with address and description
const dummy = new vyos.InterfaceDummy("dum0", {
    name: "dum0",
    description: "Management network",
    address: ["10.0.0.1/24"],
    mtu: 1500,
});

// Firewall address group
new vyos.FirewallGroupAddressGroup("web-servers", {
    name: "web-servers",
    description: "Web server pool",
    address: ["10.0.1.10", "10.0.1.11", "10.0.1.12"],
});

// NTP server
new vyos.ServiceNTPServer("pool-ntp", {
    name: "pool.ntp.org",
    pool: true,
    prefer: true,
});

// Component resource: static route with next-hops
const mgmtRoute = new vyos.StaticRouteComplete("mgmt-route", {
    prefix: "10.10.0.0/16",
    description: "Management network route",
    nextHops: [
        { address: "10.0.0.254" },
        { address: "10.0.0.253", distance: 10 },
    ],
});

// Component resource: firewall ruleset with rules
const allowWeb = new vyos.FirewallIPv4Ruleset("allow-web", {
    name: "allow-web",
    defaultAction: "drop",
    description: "Allow inbound web traffic",
    rules: [
        { number: 10, action: "accept", protocol: "tcp", destinationPort: "80", state: ["established", "new"] },
        { number: 20, action: "accept", protocol: "tcp", destinationPort: "443", state: ["established", "new"] },
    ],
});

// NAT masquerade rule
new vyos.NATSourceRule("masquerade", {
    name: "100",
    outboundInterfaceName: "eth0",
    translationAddress: "masquerade",
});

// Save config to disk
new vyos.ConfigFileSave("save-config", {
    triggers: { version: "1" },
});

// Exports
export const hostnameValue = hostname.hostName;
export const dummyAddress = dummy.address;
export const routePrefix = mgmtRoute.prefix;
export const firewallName = allowWeb.name;
