import pulumi
import pulumi_vyos as vyos

# System hostname
hostname = vyos.SystemHostName("hostname", host_name="my-vyos-router")

# Dummy interface with address and description
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

# Component resource: static route with next-hops (no manual depends_on needed)
mgmt_route = vyos.StaticRouteComplete("mgmt-route",
    prefix="10.10.0.0/16",
    description="Management network route",
    next_hops=[
        vyos.NextHopArgsArgs(address="10.0.0.254"),
        vyos.NextHopArgsArgs(address="10.0.0.253", distance=10),
    ],
)

# Component resource: firewall ruleset with rules
allow_web = vyos.FirewallIPv4Ruleset("allow-web",
    name="allow-web",
    default_action="drop",
    description="Allow inbound web traffic",
    rules=[
        vyos.FirewallRuleArgsArgs(number=10, action="accept", protocol="tcp",
                                  destination_port="80", state=["established", "new"]),
        vyos.FirewallRuleArgsArgs(number=20, action="accept", protocol="tcp",
                                  destination_port="443", state=["established", "new"]),
    ],
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

# Exports
pulumi.export("hostname", hostname.host_name)
pulumi.export("dummy_address", dummy.address)
pulumi.export("route_prefix", mgmt_route.prefix)
pulumi.export("firewall_name", allow_web.name)
