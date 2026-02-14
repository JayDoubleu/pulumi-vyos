package provider

import (
	"fmt"
	"strconv"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// FirewallIPv4RulesetArgs defines the inputs for the FirewallIPv4Ruleset component.
type FirewallIPv4RulesetArgs struct {
	Name          string             `pulumi:"name"`
	DefaultAction *string            `pulumi:"defaultAction,optional"`
	DefaultLog    *bool              `pulumi:"defaultLog,optional"`
	Description   *string            `pulumi:"description,optional"`
	Rules         []FirewallRuleArgs `pulumi:"rules"`
}

// FirewallRuleArgs defines a single firewall rule within a FirewallIPv4Ruleset.
type FirewallRuleArgs struct {
	Number             int      `pulumi:"number"`
	Action             string   `pulumi:"action"`
	Protocol           *string  `pulumi:"protocol,optional"`
	Description        *string  `pulumi:"description,optional"`
	Log                *bool    `pulumi:"log,optional"`
	Disable            *bool    `pulumi:"disable,optional"`
	State              []string `pulumi:"state,optional"`
	SourceAddress      *string  `pulumi:"sourceAddress,optional"`
	SourcePort         *string  `pulumi:"sourcePort,optional"`
	DestinationAddress *string  `pulumi:"destinationAddress,optional"`
	DestinationPort    *string  `pulumi:"destinationPort,optional"`
}

// FirewallIPv4Ruleset bundles a named IPv4 firewall policy with its rules into
// a single component resource. This covers the most common rule fields; for
// advanced fields (TCP flags, GRE, synproxy, etc.) use the underlying
// FirewallIPv4NameRule resource directly.
type FirewallIPv4Ruleset struct {
	pulumi.ResourceState

	Name      pulumi.StringOutput `pulumi:"name"`
	RuleCount pulumi.IntOutput    `pulumi:"ruleCount"`
}

// Annotate provides schema metadata for the FirewallIPv4Ruleset component.
func (c *FirewallIPv4Ruleset) Annotate(a infer.Annotator) {
	a.Describe(c, "A component resource that creates an IPv4 firewall ruleset with its "+
		"rules. Child resources are automatically parented and ordered.")
	a.Describe(&c.Name, "The firewall ruleset name.")
	a.Describe(&c.RuleCount, "The number of rules created in this ruleset.")
}

// Annotate provides schema metadata for the FirewallIPv4Ruleset inputs.
func (args *FirewallIPv4RulesetArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Name, "Firewall ruleset name (alphanumeric, hyphens, dots).")
	a.Describe(&args.DefaultAction, "Default action for the ruleset (drop, accept, reject, jump, return, continue).")
	a.Describe(&args.DefaultLog, "Log packets hitting the default action.")
	a.Describe(&args.Description, "Description for the ruleset.")
	a.Describe(&args.Rules, "Firewall rules to create in this ruleset.")
}

// Annotate provides schema metadata for the FirewallRuleArgs type.
func (r *FirewallRuleArgs) Annotate(a infer.Annotator) {
	a.Describe(&r.Number, "Rule number (1-999999).")
	a.Describe(&r.Action, "Rule action (accept, drop, reject, jump, return, continue).")
	a.Describe(&r.Protocol, "Protocol to match (tcp, udp, icmp, all, etc.).")
	a.Describe(&r.Description, "Rule description.")
	a.Describe(&r.Log, "Log packets matching this rule.")
	a.Describe(&r.Disable, "Disable this rule.")
	a.Describe(&r.State, "Connection states to match (established, new, related, invalid).")
	a.Describe(&r.SourceAddress, "Source address or network.")
	a.Describe(&r.SourcePort, "Source port or port range.")
	a.Describe(&r.DestinationAddress, "Destination address or network.")
	a.Describe(&r.DestinationPort, "Destination port or port range.")
}

// NewFirewallIPv4Ruleset constructs a FirewallIPv4Ruleset component resource.
func NewFirewallIPv4Ruleset(
	ctx *pulumi.Context, name string, args FirewallIPv4RulesetArgs, opts ...pulumi.ResourceOption,
) (*FirewallIPv4Ruleset, error) {
	comp := &FirewallIPv4Ruleset{}
	err := ctx.RegisterComponentResource(p.GetTypeToken(ctx), name, comp, opts...)
	if err != nil {
		return nil, err
	}

	// Build the parent firewall name properties.
	fwProps := pulumi.Map{
		"name": pulumi.String(args.Name),
	}
	if args.DefaultAction != nil {
		fwProps["defaultAction"] = pulumi.String(*args.DefaultAction)
	}
	if args.DefaultLog != nil {
		fwProps["defaultLog"] = pulumi.Bool(*args.DefaultLog)
	}
	if args.Description != nil {
		fwProps["description"] = pulumi.String(*args.Description)
	}

	fw := &childResource{}
	err = ctx.RegisterResource("vyos:index:FirewallIPv4Name", name+"-fw", fwProps, fw,
		pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("creating firewall name: %w", err)
	}

	// Create a rule child resource for each entry.
	for _, rule := range args.Rules {
		ruleProps := pulumi.Map{
			"nameName": pulumi.String(args.Name),
			"name":     pulumi.String(strconv.Itoa(rule.Number)),
			"action":   pulumi.String(rule.Action),
		}
		if rule.Protocol != nil {
			ruleProps["protocol"] = pulumi.String(*rule.Protocol)
		}
		if rule.Description != nil {
			ruleProps["description"] = pulumi.String(*rule.Description)
		}
		if rule.Log != nil {
			ruleProps["log"] = pulumi.Bool(*rule.Log)
		}
		if rule.Disable != nil {
			ruleProps["disable"] = pulumi.Bool(*rule.Disable)
		}
		if len(rule.State) > 0 {
			states := make(pulumi.StringArray, len(rule.State))
			for i, s := range rule.State {
				states[i] = pulumi.String(s)
			}
			ruleProps["state"] = states
		}
		if rule.SourceAddress != nil {
			ruleProps["sourceAddress"] = pulumi.String(*rule.SourceAddress)
		}
		if rule.SourcePort != nil {
			ruleProps["sourcePort"] = pulumi.String(*rule.SourcePort)
		}
		if rule.DestinationAddress != nil {
			ruleProps["destinationAddress"] = pulumi.String(*rule.DestinationAddress)
		}
		if rule.DestinationPort != nil {
			ruleProps["destinationPort"] = pulumi.String(*rule.DestinationPort)
		}

		ruleResource := &childResource{}
		ruleName := fmt.Sprintf("%s-rule-%d", name, rule.Number)
		err = ctx.RegisterResource("vyos:index:FirewallIPv4NameRule", ruleName, ruleProps, ruleResource,
			pulumi.Parent(comp), pulumi.DependsOn([]pulumi.Resource{fw}))
		if err != nil {
			return nil, fmt.Errorf("creating rule %d: %w", rule.Number, err)
		}
	}

	comp.Name = pulumi.String(args.Name).ToStringOutput()
	comp.RuleCount = pulumi.Int(len(args.Rules)).ToIntOutput()

	return comp, nil
}
