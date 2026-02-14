package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/codegen/model"
)

func TestGenerate_TagNodeResource(t *testing.T) {
	t.Parallel()

	resources := []model.Resource{
		{
			GoName:      "InterfaceEthernet",
			Description: "Ethernet Interface",
			Kind:        model.TagNodeResource,
			FileName:    "resource_gen_interface_ethernet.go",
			TagFields: []model.TagField{
				{
					GoName:      "Name",
					PulumiName:  "name",
					Description: "Ethernet Interface",
					PathPrefix:  []string{"interfaces", "ethernet"},
					Constraint: &model.FieldConstraint{
						Patterns:     []string{`((eth|lan)[0-9]+|(eno|ens|enp|enx).+)`},
						ErrorMessage: "Invalid Ethernet interface name",
					},
				},
			},
			Fields: []model.Field{
				{
					GoName:      "Description",
					PulumiName:  "description",
					VyosName:    "description",
					VyosPath:    []string{"description"},
					GoType:      "*string",
					FieldType:   model.StringField,
					Description: "Description",
				},
				{
					GoName:      "Disable",
					PulumiName:  "disable",
					VyosName:    "disable",
					VyosPath:    []string{"disable"},
					GoType:      "*bool",
					FieldType:   model.BoolField,
					Description: "Disable interface",
				},
				{
					GoName:      "MTU",
					PulumiName:  "mtu",
					VyosName:    "mtu",
					VyosPath:    []string{"mtu"},
					GoType:      "*int",
					FieldType:   model.IntField,
					Description: "MTU",
					Constraint: &model.FieldConstraint{
						NumericRange: &model.Range{Min: 68, Max: 16000},
						ErrorMessage: "MTU must be between 68 and 16000",
					},
				},
				{
					GoName:      "Address",
					PulumiName:  "address",
					VyosName:    "address",
					VyosPath:    []string{"address"},
					GoType:      "[]string",
					FieldType:   model.MultiField,
					Description: "IP address",
				},
				{
					GoName:      "OffloadGRO",
					PulumiName:  "offloadGRO",
					VyosName:    "gro",
					VyosPath:    []string{"offload", "gro"},
					GoType:      "*bool",
					FieldType:   model.BoolField,
					Description: "Enable GRO",
				},
				{
					GoName:      "Duplex",
					PulumiName:  "duplex",
					VyosName:    "duplex",
					VyosPath:    []string{"duplex"},
					GoType:      "*string",
					FieldType:   model.StringField,
					Description: "Duplex mode",
					Constraint: &model.FieldConstraint{
						Patterns:     []string{`(auto|half|full)`},
						ErrorMessage: "duplex must be auto, half or full",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	if err := Generate(resources, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check that all files were created.
	for _, name := range []string{
		"resource_gen_helpers.go",
		"resource_gen_interface_ethernet.go",
		"resource_gen_registration.go",
	} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}

	// Verify resource file content.
	data, err := os.ReadFile(filepath.Join(dir, "resource_gen_interface_ethernet.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Check for patterns that survive gofmt alignment (avoid space-sensitive
	// patterns in struct fields since gofmt uses tabs for alignment).
	for _, want := range []string{
		"type InterfaceEthernet struct{}",
		"type InterfaceEthernetArgs struct",
		`pulumi:"name"`,
		`pulumi:"description,optional"`,
		`pulumi:"disable,optional"`,
		`pulumi:"mtu,optional"`,
		`pulumi:"address,optional"`,
		`pulumi:"offloadGRO,optional"`,
		"func interfaceEthernetBasePath(name string)",
		`"interfaces", "ethernet", name`,
		"func (InterfaceEthernet) Create(",
		"func (InterfaceEthernet) Read(",
		"func (InterfaceEthernet) Update(",
		"func (InterfaceEthernet) Delete(",
		"func buildInterfaceEthernetOps(",
		"func buildInterfaceEthernetUpdateOps(",
		"func parseInterfaceEthernetConfig(",
		`"strconv"`,
		`strconv.Itoa(*args.MTU)`,
		`getNestedValue(data, []string{"offload", "gro"})`,
		"saveIfEnabled(ctx)",
		// Check method and constraint validation.
		`"regexp"`,
		"func (InterfaceEthernet) Check(",
		"interfaceEthernetNameRe",
		"regexp.MustCompile",
		"checkRegex(inputs.Name",
		"checkIntRange(int64(*inputs.MTU)",
		"68, 16000",
		"checkRegex(*inputs.Duplex",
		"interfaceEthernetDuplexRe",
		"infer.DefaultCheck[InterfaceEthernetArgs]",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in generated code", want)
		}
	}

	// Verify registration file.
	regData, err := os.ReadFile(filepath.Join(dir, "resource_gen_registration.go"))
	if err != nil {
		t.Fatal(err)
	}
	regContent := string(regData)
	if !strings.Contains(regContent, "infer.Resource(InterfaceEthernet{})") {
		t.Error("registration missing InterfaceEthernet")
	}
}

func TestGenerate_LeafNodeResource(t *testing.T) {
	t.Parallel()

	resources := []model.Resource{
		{
			GoName:      "SystemHostName",
			Description: "System host name",
			Kind:        model.LeafNodeResource,
			FileName:    "resource_gen_system_host_name.go",
			LeafPath:    []string{"system", "host-name"},
			Fields: []model.Field{
				{
					GoName:      "HostName",
					PulumiName:  "hostName",
					VyosName:    "host-name",
					VyosPath:    []string{"host-name"},
					GoType:      "string",
					FieldType:   model.StringField,
					Description: "System host name",
				},
			},
		},
	}

	dir := t.TempDir()
	if err := Generate(resources, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "resource_gen_system_host_name.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"type SystemHostName struct{}",
		"type SystemHostNameArgs struct",
		// Required field (no ,optional tag).
		`pulumi:"hostName"`,
		// Leaf CRUD patterns.
		`client.Set(ctx, []string{"system", "host-name"}`,
		`client.Delete(ctx, []string{"system", "host-name"}`,
		`client.ShowConfig(ctx, []string{"system"})`,
		`raw["host-name"]`,
		"saveIfEnabled(ctx)",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in generated code", want)
		}
	}

	// Verify no ,optional on the required field.
	if strings.Contains(content, `pulumi:"hostName,optional"`) {
		t.Error("leaf field should not be optional")
	}
}

func TestGenerate_NestedTagNodeResource(t *testing.T) {
	t.Parallel()

	resources := []model.Resource{
		{
			GoName:      "FirewallZoneFrom",
			Description: "Zone from which to filter traffic",
			Kind:        model.TagNodeResource,
			FileName:    "resource_gen_firewall_zone_from.go",
			TagFields: []model.TagField{
				{
					GoName:      "ZoneName",
					PulumiName:  "zoneName",
					Description: "Parent tag node key",
					PathPrefix:  []string{"firewall", "zone"},
				},
				{
					GoName:      "Name",
					PulumiName:  "name",
					Description: "Zone from which to filter traffic",
					PathPrefix:  []string{"from"},
				},
			},
			Fields: []model.Field{
				{
					GoName:      "FirewallName",
					PulumiName:  "firewallName",
					VyosName:    "name",
					VyosPath:    []string{"firewall", "name"},
					GoType:      "*string",
					FieldType:   model.StringField,
					Description: "IPv4 firewall ruleset",
				},
			},
		},
	}

	dir := t.TempDir()
	if err := Generate(resources, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "resource_gen_firewall_zone_from.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		// Two tag fields (check struct tags, not field declarations).
		`pulumi:"zoneName"`,
		`pulumi:"name"`,
		// basePath with two parameters.
		"func firewallZoneFromBasePath(zoneName, name string)",
		`"firewall", "zone", zoneName, "from", name`,
		// Read uses req.State for both tags.
		"req.State.ZoneName, req.State.Name",
		// buildOps uses args for both tags.
		"args.ZoneName, args.Name",
		// buildUpdateOps uses cur for both tags.
		"cur.ZoneName, cur.Name",
		// parseConfig receives both tag values as params.
		"func parseFirewallZoneFromConfig(zoneName, name string,",
		"ZoneName: zoneName,",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in generated code", want)
		}
	}
}

func TestTemplateData_LowerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		goName string
		want   string
	}{
		{"InterfaceEthernet", "interfaceEthernet"},
		{"FirewallZoneFrom", "firewallZoneFrom"},
		{"SystemHostName", "systemHostName"},
		{"MTU", "mtu"},
	}

	for _, tt := range tests {
		td := NewTemplateData(&model.Resource{GoName: tt.goName})
		if got := td.LowerName(); got != tt.want {
			t.Errorf("LowerName(%q) = %q, want %q", tt.goName, got, tt.want)
		}
	}
}

func TestVyosPathLit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path []string
		want string
	}{
		{[]string{"address"}, `"address"`},
		{[]string{"offload", "gro"}, `"offload", "gro"`},
		{[]string{"system", "host-name"}, `"system", "host-name"`},
	}

	for _, tt := range tests {
		if got := vyosPathLit(tt.path); got != tt.want {
			t.Errorf("vyosPathLit(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
