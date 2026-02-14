package main

import (
	"github.com/jaydoubleu/pulumi-vyos/sdk/go/vyos"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// System hostname
		hostname, err := vyos.NewSystemHostName(ctx, "hostname", &vyos.SystemHostNameArgs{
			HostName: pulumi.String("my-vyos-router"),
		})
		if err != nil {
			return err
		}

		// Dummy interface with address and description
		dummy, err := vyos.NewInterfaceDummy(ctx, "dum0", &vyos.InterfaceDummyArgs{
			Name:        pulumi.String("dum0"),
			Description: pulumi.StringRef("Management network"),
			Address:     pulumi.ToStringArray([]string{"10.0.0.1/24"}),
			Mtu:         pulumi.IntRef(1500),
		})
		if err != nil {
			return err
		}

		// Firewall address group
		_, err = vyos.NewFirewallGroupAddressGroup(ctx, "web-servers", &vyos.FirewallGroupAddressGroupArgs{
			Name:        pulumi.String("web-servers"),
			Description: pulumi.StringRef("Web server pool"),
			Address:     pulumi.ToStringArray([]string{"10.0.1.10", "10.0.1.11", "10.0.1.12"}),
		})
		if err != nil {
			return err
		}

		// NTP server
		_, err = vyos.NewServiceNTPServer(ctx, "pool-ntp", &vyos.ServiceNTPServerArgs{
			Name:   pulumi.String("pool.ntp.org"),
			Pool:   pulumi.BoolRef(true),
			Prefer: pulumi.BoolRef(true),
		})
		if err != nil {
			return err
		}

		// Component resource: static route with next-hops
		mgmtRoute, err := vyos.NewStaticRouteComplete(ctx, "mgmt-route", &vyos.StaticRouteCompleteArgs{
			Prefix:      pulumi.String("10.10.0.0/16"),
			Description: pulumi.StringRef("Management network route"),
			NextHops: vyos.NextHopArgsArray{
				vyos.NextHopArgsArgs{Address: pulumi.String("10.0.0.254")},
				vyos.NextHopArgsArgs{Address: pulumi.String("10.0.0.253"), Distance: pulumi.IntRef(10)},
			},
		})
		if err != nil {
			return err
		}

		// Component resource: firewall ruleset with rules
		allowWeb, err := vyos.NewFirewallIPv4Ruleset(ctx, "allow-web", &vyos.FirewallIPv4RulesetArgs{
			Name:          pulumi.String("allow-web"),
			DefaultAction: pulumi.StringRef("drop"),
			Description:   pulumi.StringRef("Allow inbound web traffic"),
			Rules: vyos.FirewallRuleArgsArray{
				vyos.FirewallRuleArgsArgs{
					Number:          pulumi.Int(10),
					Action:          pulumi.String("accept"),
					Protocol:        pulumi.StringRef("tcp"),
					DestinationPort: pulumi.StringRef("80"),
					State:           pulumi.ToStringArray([]string{"established", "new"}),
				},
				vyos.FirewallRuleArgsArgs{
					Number:          pulumi.Int(20),
					Action:          pulumi.String("accept"),
					Protocol:        pulumi.StringRef("tcp"),
					DestinationPort: pulumi.StringRef("443"),
					State:           pulumi.ToStringArray([]string{"established", "new"}),
				},
			},
		})
		if err != nil {
			return err
		}

		// NAT masquerade rule
		_, err = vyos.NewNATSourceRule(ctx, "masquerade", &vyos.NATSourceRuleArgs{
			Name:                  pulumi.String("100"),
			OutboundInterfaceName: pulumi.StringRef("eth0"),
			TranslationAddress:   pulumi.StringRef("masquerade"),
		})
		if err != nil {
			return err
		}

		// Save config to disk
		_, err = vyos.NewConfigFileSave(ctx, "save-config", &vyos.ConfigFileSaveArgs{
			Triggers: pulumi.StringMap{"version": pulumi.String("1")},
		})
		if err != nil {
			return err
		}

		// Exports
		ctx.Export("hostname", hostname.HostName)
		ctx.Export("dummyAddress", dummy.Address)
		ctx.Export("routePrefix", mgmtRoute.Prefix)
		ctx.Export("firewallName", allowWeb.Name)
		return nil
	})
}
