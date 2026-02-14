package model

import (
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

func TestBuild_SystemHostName(t *testing.T) {
	t.Parallel()

	def := &xmlparse.InterfaceDefinition{
		Nodes: []xmlparse.Node{
			{
				Name: "system",
				Children: &xmlparse.Children{
					LeafNodes: []xmlparse.LeafNode{
						{
							Name:  "host-name",
							Owner: "system_host-name.py",
							Properties: &xmlparse.Properties{
								Help: "System host name",
							},
						},
					},
				},
			},
		},
	}

	resources := Build(def)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r := resources[0]
	if r.GoName != "SystemHostName" {
		t.Errorf("GoName = %q, want %q", r.GoName, "SystemHostName")
	}
	if r.Kind != LeafNodeResource {
		t.Errorf("Kind = %d, want LeafNodeResource", r.Kind)
	}
	if len(r.LeafPath) != 2 || r.LeafPath[0] != "system" || r.LeafPath[1] != "host-name" {
		t.Errorf("LeafPath = %v, want [system host-name]", r.LeafPath)
	}
	if len(r.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(r.Fields))
	}
	if r.Fields[0].GoName != "HostName" {
		t.Errorf("field GoName = %q, want %q", r.Fields[0].GoName, "HostName")
	}
	if r.Fields[0].FieldType != StringField {
		t.Errorf("field type = %d, want StringField", r.Fields[0].FieldType)
	}
	if r.Fields[0].GoType != "string" {
		t.Errorf("field GoType = %q, want %q", r.Fields[0].GoType, "string")
	}
}

func TestBuild_InterfaceEthernet(t *testing.T) {
	t.Parallel()

	def := &xmlparse.InterfaceDefinition{
		Nodes: []xmlparse.Node{
			{
				Name: "interfaces",
				Children: &xmlparse.Children{
					TagNodes: []xmlparse.TagNode{
						{
							Name:  "ethernet",
							Owner: "interfaces_ethernet.py",
							Properties: &xmlparse.Properties{
								Help: "Ethernet Interface",
							},
							Children: &xmlparse.Children{
								LeafNodes: []xmlparse.LeafNode{
									{
										Name: "description",
										Properties: &xmlparse.Properties{
											Help: "Description",
										},
									},
									{
										Name: "disable",
										Properties: &xmlparse.Properties{
											Help:      "Disable interface",
											Valueless: &struct{}{},
										},
									},
									{
										Name: "mtu",
										Properties: &xmlparse.Properties{
											Help: "MTU",
											ValueHelps: []xmlparse.ValueHelp{
												{Format: "u32:68-16000"},
											},
										},
									},
									{
										Name: "address",
										Properties: &xmlparse.Properties{
											Help:  "IP address",
											Multi: &struct{}{},
										},
									},
								},
								Nodes: []xmlparse.Node{
									{
										Name: "offload",
										Properties: &xmlparse.Properties{
											Help: "Configurable offload options",
										},
										Children: &xmlparse.Children{
											LeafNodes: []xmlparse.LeafNode{
												{
													Name: "gro",
													Properties: &xmlparse.Properties{
														Help:      "Enable GRO",
														Valueless: &struct{}{},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resources := Build(def)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d: %v", len(resources), resourceNames(resources))
	}

	r := resources[0]
	if r.GoName != "InterfaceEthernet" {
		t.Errorf("GoName = %q, want %q", r.GoName, "InterfaceEthernet")
	}
	if r.Kind != TagNodeResource {
		t.Errorf("Kind = %d, want TagNodeResource", r.Kind)
	}

	// Check TagFields
	if len(r.TagFields) != 1 {
		t.Fatalf("expected 1 TagField, got %d", len(r.TagFields))
	}
	if r.TagFields[0].GoName != "Name" {
		t.Errorf("TagField GoName = %q, want %q", r.TagFields[0].GoName, "Name")
	}

	// Check fields
	fieldMap := make(map[string]Field)
	for _, f := range r.Fields {
		fieldMap[f.GoName] = f
	}

	checkField(t, fieldMap, "Description", StringField, "*string", []string{"description"})
	checkField(t, fieldMap, "Disable", BoolField, "*bool", []string{"disable"})
	checkField(t, fieldMap, "MTU", IntField, "*int", []string{"mtu"})
	checkField(t, fieldMap, "Address", MultiField, "[]string", []string{"address"})
	checkField(t, fieldMap, "OffloadGRO", BoolField, "*bool", []string{"offload", "gro"})
}

func TestBuild_FirewallWithSubResources(t *testing.T) {
	t.Parallel()

	def := &xmlparse.InterfaceDefinition{
		Nodes: []xmlparse.Node{
			{
				Name:  "firewall",
				Owner: "firewall.py",
				Properties: &xmlparse.Properties{
					Help: "Firewall",
				},
				Children: &xmlparse.Children{
					TagNodes: []xmlparse.TagNode{
						{
							Name: "flowtable",
							Properties: &xmlparse.Properties{
								Help: "Flowtable",
							},
							Children: &xmlparse.Children{
								LeafNodes: []xmlparse.LeafNode{
									{
										Name: "offload",
										Properties: &xmlparse.Properties{
											Help: "Offloading method",
										},
									},
								},
							},
						},
						{
							Name: "zone",
							Properties: &xmlparse.Properties{
								Help: "Zone-policy",
							},
							Children: &xmlparse.Children{
								LeafNodes: []xmlparse.LeafNode{
									{
										Name: "default-action",
										Properties: &xmlparse.Properties{
											Help: "Default-action",
										},
									},
									{
										Name: "local-zone",
										Properties: &xmlparse.Properties{
											Help:      "Zone to be local-zone",
											Valueless: &struct{}{},
										},
									},
								},
								TagNodes: []xmlparse.TagNode{
									{
										Name: "from",
										Properties: &xmlparse.Properties{
											Help: "Zone from which to filter traffic",
										},
										Children: &xmlparse.Children{
											Nodes: []xmlparse.Node{
												{
													Name: "firewall",
													Children: &xmlparse.Children{
														LeafNodes: []xmlparse.LeafNode{
															{
																Name: "name",
																Properties: &xmlparse.Properties{
																	Help: "IPv4 firewall ruleset",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Nodes: []xmlparse.Node{
						{
							Name: "group",
							Children: &xmlparse.Children{
								TagNodes: []xmlparse.TagNode{
									{
										Name: "address-group",
										Properties: &xmlparse.Properties{
											Help: "Firewall address-group",
										},
										Children: &xmlparse.Children{
											LeafNodes: []xmlparse.LeafNode{
												{
													Name: "address",
													Properties: &xmlparse.Properties{
														Help:  "Address-group member",
														Multi: &struct{}{},
													},
												},
												{
													Name: "description",
													Properties: &xmlparse.Properties{
														Help: "Description",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resources := Build(def)
	names := resourceNames(resources)

	// Expected resources: FirewallFlowtable, FirewallZone, FirewallZoneFrom, FirewallGroupAddressGroup
	expected := []string{"FirewallFlowtable", "FirewallZone", "FirewallZoneFrom", "FirewallGroupAddressGroup"}
	if len(resources) != len(expected) {
		t.Fatalf("expected %d resources, got %d: %v", len(expected), len(resources), names)
	}

	rMap := make(map[string]*Resource)
	for i := range resources {
		rMap[resources[i].GoName] = &resources[i]
	}

	for _, name := range expected {
		if _, ok := rMap[name]; !ok {
			t.Errorf("missing expected resource %q in %v", name, names)
		}
	}

	// Check FirewallZoneFrom has two TagFields (parent zone + own name).
	zf := rMap["FirewallZoneFrom"]
	if zf == nil {
		t.Fatal("FirewallZoneFrom not found")
	}
	if len(zf.TagFields) != 2 {
		t.Fatalf("expected 2 TagFields, got %d", len(zf.TagFields))
	}
	if zf.TagFields[0].GoName != "ZoneName" {
		t.Errorf("first TagField = %q, want %q", zf.TagFields[0].GoName, "ZoneName")
	}
	if zf.TagFields[1].GoName != "Name" {
		t.Errorf("second TagField = %q, want %q", zf.TagFields[1].GoName, "Name")
	}

	// Check FirewallZoneFrom has a nested field "FirewallName".
	zfFieldMap := make(map[string]Field)
	for _, f := range zf.Fields {
		zfFieldMap[f.GoName] = f
	}
	if _, ok := zfFieldMap["FirewallName"]; !ok {
		t.Errorf("FirewallZoneFrom missing FirewallName field, has: %v", fieldNames(zf.Fields))
	}

	// Check FirewallGroupAddressGroup fields.
	ag := rMap["FirewallGroupAddressGroup"]
	if ag == nil {
		t.Fatal("FirewallGroupAddressGroup not found")
	}
	agFieldMap := make(map[string]Field)
	for _, f := range ag.Fields {
		agFieldMap[f.GoName] = f
	}
	checkField(t, agFieldMap, "Address", MultiField, "[]string", []string{"address"})
	checkField(t, agFieldMap, "Description", StringField, "*string", []string{"description"})
}

func TestBuild_NestedTagNodes(t *testing.T) {
	t.Parallel()

	// protocols > static (owned node) > mroute (tagNode) > next-hop (nested tagNode)
	def := &xmlparse.InterfaceDefinition{
		Nodes: []xmlparse.Node{
			{
				Name: "protocols",
				Children: &xmlparse.Children{
					Nodes: []xmlparse.Node{
						{
							Name:  "static",
							Owner: "protocols_static.py",
							Properties: &xmlparse.Properties{
								Help: "Static Routing",
							},
							Children: &xmlparse.Children{
								TagNodes: []xmlparse.TagNode{
									{
										Name: "mroute",
										Properties: &xmlparse.Properties{
											Help: "Multicast route",
										},
										Children: &xmlparse.Children{
											TagNodes: []xmlparse.TagNode{
												{
													Name: "next-hop",
													Properties: &xmlparse.Properties{
														Help: "Next-hop router",
													},
													Children: &xmlparse.Children{
														LeafNodes: []xmlparse.LeafNode{
															{
																Name: "distance",
																Properties: &xmlparse.Properties{
																	Help: "Distance",
																	ValueHelps: []xmlparse.ValueHelp{
																		{Format: "u32:1-255"},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resources := Build(def)
	names := resourceNames(resources)

	// Expected: ProtocolStaticMroute, ProtocolStaticMrouteNextHop
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d: %v", len(resources), names)
	}

	rMap := make(map[string]*Resource)
	for i := range resources {
		rMap[resources[i].GoName] = &resources[i]
	}

	if _, ok := rMap["ProtocolStaticMroute"]; !ok {
		t.Errorf("missing ProtocolStaticMroute in %v", names)
	}

	nh := rMap["ProtocolStaticMrouteNextHop"]
	if nh == nil {
		t.Fatalf("missing ProtocolStaticMrouteNextHop in %v", names)
	}

	// next-hop has 2 tag fields: MrouteName (parent) + Name (own)
	if len(nh.TagFields) != 2 {
		t.Fatalf("expected 2 TagFields, got %d", len(nh.TagFields))
	}

	// Check distance field is IntField
	if len(nh.Fields) != 1 || nh.Fields[0].GoName != "Distance" || nh.Fields[0].FieldType != IntField {
		t.Errorf("unexpected fields: %v", fieldNames(nh.Fields))
	}
}

func checkField(t *testing.T, fields map[string]Field, name string, ft FieldType, goType string, vyosPath []string) {
	t.Helper()
	f, ok := fields[name]
	if !ok {
		t.Errorf("field %q not found", name)
		return
	}
	if f.FieldType != ft {
		t.Errorf("field %q type = %d, want %d", name, f.FieldType, ft)
	}
	if f.GoType != goType {
		t.Errorf("field %q GoType = %q, want %q", name, f.GoType, goType)
	}
	if len(f.VyosPath) != len(vyosPath) {
		t.Errorf("field %q VyosPath = %v, want %v", name, f.VyosPath, vyosPath)
		return
	}
	for i := range vyosPath {
		if f.VyosPath[i] != vyosPath[i] {
			t.Errorf("field %q VyosPath[%d] = %q, want %q", name, i, f.VyosPath[i], vyosPath[i])
		}
	}
}

func resourceNames(resources []Resource) []string {
	names := make([]string, len(resources))
	for i, r := range resources {
		names[i] = r.GoName
	}
	return names
}

func fieldNames(fields []Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.GoName
	}
	return names
}
