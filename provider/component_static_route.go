package provider

import (
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// childResource is a minimal custom resource state holder for child resources
// created by component constructors.
type childResource struct {
	pulumi.CustomResourceState
}

// StaticRouteCompleteArgs defines the inputs for the StaticRouteComplete component.
type StaticRouteCompleteArgs struct {
	Prefix      string        `pulumi:"prefix"`
	Description *string       `pulumi:"description,optional"`
	NextHops    []NextHopArgs `pulumi:"nextHops"`
}

// NextHopArgs defines a single next-hop entry within a StaticRouteComplete.
type NextHopArgs struct {
	Address   string  `pulumi:"address"`
	Distance  *int    `pulumi:"distance,optional"`
	Interface *string `pulumi:"interface,optional"`
	Disable   *bool   `pulumi:"disable,optional"`
	VRF       *string `pulumi:"vrf,optional"`
}

// StaticRouteComplete bundles a static IPv4 route with one or more next-hops
// into a single component resource. This avoids the need for manual depends_on
// between the route and its next-hop resources.
type StaticRouteComplete struct {
	pulumi.ResourceState

	Prefix   pulumi.StringOutput      `pulumi:"prefix"`
	NextHops pulumi.StringArrayOutput `pulumi:"nextHops"`
}

// Annotate provides schema metadata for the StaticRouteComplete component.
func (c *StaticRouteComplete) Annotate(a infer.Annotator) {
	a.Describe(c, "A component resource that creates a static IPv4 route with one or more "+
		"next-hops. Child resources are automatically parented and ordered.")
	a.Describe(&c.Prefix, "The route prefix CIDR.")
	a.Describe(&c.NextHops, "The next-hop addresses created for this route.")
}

// Annotate provides schema metadata for the StaticRouteComplete inputs.
func (args *StaticRouteCompleteArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Prefix, "Static IPv4 route prefix in CIDR notation (e.g. 10.10.0.0/16).")
	a.Describe(&args.Description, "Description for the route.")
	a.Describe(&args.NextHops, "One or more next-hop entries for this route.")
}

// Annotate provides schema metadata for the NextHopArgs type.
func (nh *NextHopArgs) Annotate(a infer.Annotator) {
	a.Describe(&nh.Address, "Next-hop IPv4 router address.")
	a.Describe(&nh.Distance, "Administrative distance (1-255).")
	a.Describe(&nh.Interface, "Outbound interface name.")
	a.Describe(&nh.Disable, "Disable this next-hop.")
	a.Describe(&nh.VRF, "VRF to leak route into.")
}

// NewStaticRouteComplete constructs a StaticRouteComplete component resource.
func NewStaticRouteComplete(
	ctx *pulumi.Context, name string, args StaticRouteCompleteArgs, opts ...pulumi.ResourceOption,
) (*StaticRouteComplete, error) {
	comp := &StaticRouteComplete{}
	err := ctx.RegisterComponentResource(p.GetTypeToken(ctx), name, comp, opts...)
	if err != nil {
		return nil, err
	}

	// Build the parent route properties.
	routeProps := pulumi.Map{
		"name": pulumi.String(args.Prefix),
	}
	if args.Description != nil {
		routeProps["description"] = pulumi.String(*args.Description)
	}

	route := &childResource{}
	err = ctx.RegisterResource("vyos:index:ProtocolStaticRoute", name+"-route", routeProps, route,
		pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("creating route: %w", err)
	}

	// Create a next-hop child resource for each entry.
	nhAddresses := make(pulumi.StringArray, 0, len(args.NextHops))
	for _, nh := range args.NextHops {
		nhProps := pulumi.Map{
			"routeName": pulumi.String(args.Prefix),
			"name":      pulumi.String(nh.Address),
		}
		if nh.Distance != nil {
			nhProps["distance"] = pulumi.Int(*nh.Distance)
		}
		if nh.Interface != nil {
			nhProps["interface"] = pulumi.String(*nh.Interface)
		}
		if nh.Disable != nil {
			nhProps["disable"] = pulumi.Bool(*nh.Disable)
		}
		if nh.VRF != nil {
			nhProps["vrf"] = pulumi.String(*nh.VRF)
		}

		nhResource := &childResource{}
		nhName := fmt.Sprintf("%s-nh-%s", name, nh.Address)
		err = ctx.RegisterResource("vyos:index:ProtocolStaticRouteNextHop", nhName, nhProps, nhResource,
			pulumi.Parent(comp), pulumi.DependsOn([]pulumi.Resource{route}))
		if err != nil {
			return nil, fmt.Errorf("creating next-hop %s: %w", nh.Address, err)
		}

		nhAddresses = append(nhAddresses, pulumi.String(nh.Address))
	}

	comp.Prefix = pulumi.String(args.Prefix).ToStringOutput()
	comp.NextHops = nhAddresses.ToStringArrayOutput()

	return comp, nil
}
