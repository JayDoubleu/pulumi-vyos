package model

import (
	"testing"
)

func TestKebabToPascal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"description", "Description"},
		{"host-name", "HostName"},
		{"mtu", "MTU"},
		{"mac", "MAC"},
		{"ipv6-address", "IPv6Address"},
		{"disable-flow-control", "DisableFlowControl"},
		{"address-group", "AddressGroup"},
		{"ethernet", "Ethernet"},
		{"duplex", "Duplex"},
		{"speed", "Speed"},
		{"disable", "Disable"},
		{"hw-id", "HWID"},
		{"gro", "GRO"},
		{"tso", "TSO"},
		{"rx-usecs", "RXUsecs"},
		{"tx-frames", "TXFrames"},
		{"cqe-mode-rx", "CQEModeRX"},
		{"default-action", "DefaultAction"},
		{"local-zone", "LocalZone"},
		{"hw-tc-offload", "HWTcOffload"},
		{"vrf", "VRF"},
		{"dhcp", "DHCP"},
		{"pppoe", "PPPoE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := KebabToPascal(tt.input)
			if got != tt.want {
				t.Errorf("KebabToPascal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPascalToCamel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Description", "description"},
		{"MTU", "mtu"},
		{"MAC", "mac"},
		{"DNS", "dns"},
		{"DisableFlowControl", "disableFlowControl"},
		{"OffloadGRO", "offloadGRO"},
		{"IPv6Address", "iPv6Address"},
		{"IPAddress", "ipAddress"},
		{"Name", "name"},
		{"HostName", "hostName"},
		{"AddressGroup", "addressGroup"},
		{"HWID", "hwid"},
		{"RXUsecs", "rxUsecs"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := PascalToCamel(tt.input)
			if got != tt.want {
				t.Errorf("PascalToCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSingularizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"interfaces", "interface"},
		{"protocols", "protocol"},
		{"services", "service"},
		{"firewall", "firewall"},
		{"system", "system"},
		{"dns", "dns"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := SingularizePath(tt.input)
			if got != tt.want {
				t.Errorf("SingularizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildGoResourceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		segments    []string
		isContainer []bool
		want        string
	}{
		{
			name:        "interface ethernet",
			segments:    []string{"interfaces", "ethernet"},
			isContainer: []bool{true, false},
			want:        "InterfaceEthernet",
		},
		{
			name:        "system host-name",
			segments:    []string{"system", "host-name"},
			isContainer: []bool{true, false},
			want:        "SystemHostName",
		},
		{
			name:        "firewall zone",
			segments:    []string{"firewall", "zone"},
			isContainer: []bool{false, false},
			want:        "FirewallZone",
		},
		{
			name:        "firewall group address-group",
			segments:    []string{"firewall", "group", "address-group"},
			isContainer: []bool{false, true, false},
			want:        "FirewallGroupAddressGroup",
		},
		{
			name:        "protocols static mroute",
			segments:    []string{"protocols", "static", "mroute"},
			isContainer: []bool{true, false, false},
			want:        "ProtocolStaticMroute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildGoResourceName(tt.segments, tt.isContainer)
			if got != tt.want {
				t.Errorf("BuildGoResourceName(%v, %v) = %q, want %q",
					tt.segments, tt.isContainer, got, tt.want)
			}
		})
	}
}

func TestBuildGoFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   []string
		leafName string
		want     string
	}{
		{"top-level", nil, "description", "Description"},
		{"nested", []string{"offload"}, "gro", "OffloadGRO"},
		{"deep-nested", []string{"ring-buffer"}, "rx", "RingBufferRX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildGoFieldName(tt.prefix, tt.leafName)
			if got != tt.want {
				t.Errorf("BuildGoFieldName(%v, %q) = %q, want %q",
					tt.prefix, tt.leafName, got, tt.want)
			}
		})
	}
}

func TestBuildFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		segments    []string
		isContainer []bool
		want        string
	}{
		{
			name:        "interface ethernet",
			segments:    []string{"interfaces", "ethernet"},
			isContainer: []bool{true, false},
			want:        "resource_gen_interface_ethernet.go",
		},
		{
			name:        "system host-name",
			segments:    []string{"system", "host-name"},
			isContainer: []bool{true, false},
			want:        "resource_gen_system_host_name.go",
		},
		{
			name:        "firewall group address-group",
			segments:    []string{"firewall", "group", "address-group"},
			isContainer: []bool{false, true, false},
			want:        "resource_gen_firewall_group_address_group.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildFileName(tt.segments, tt.isContainer)
			if got != tt.want {
				t.Errorf("BuildFileName(%v, %v) = %q, want %q",
					tt.segments, tt.isContainer, got, tt.want)
			}
		})
	}
}

func TestDeduplicate_NoDuplicates(t *testing.T) {
	t.Parallel()

	resources := []Resource{
		{GoName: "InterfaceEthernet", FileName: "resource_gen_interface_ethernet.go"},
		{GoName: "SystemHostName", FileName: "resource_gen_system_host_name.go"},
	}

	got := Deduplicate(resources)
	if len(got) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(got))
	}
	if got[0].GoName != "InterfaceEthernet" || got[1].GoName != "SystemHostName" {
		t.Errorf("unexpected resources: %v, %v", got[0].GoName, got[1].GoName)
	}
}

func TestDeduplicate_SamePath(t *testing.T) {
	t.Parallel()

	// Two resources with the same GoName and same API path (duplicate XML).
	// Keep the one with more fields.
	resources := []Resource{
		{
			GoName:   "FirewallZone",
			FileName: "resource_gen_firewall_zone.go",
			Kind:     TagNodeResource,
			TagFields: []TagField{
				{PathPrefix: []string{"firewall", "zone"}},
			},
			Fields: []Field{{GoName: "Description"}},
		},
		{
			GoName:   "FirewallZone",
			FileName: "resource_gen_firewall_zone.go",
			Kind:     TagNodeResource,
			TagFields: []TagField{
				{PathPrefix: []string{"firewall", "zone"}},
			},
			Fields: []Field{{GoName: "Description"}, {GoName: "DefaultAction"}},
		},
	}

	got := Deduplicate(resources)
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}
	if len(got[0].Fields) != 2 {
		t.Errorf("expected resource with 2 fields, got %d", len(got[0].Fields))
	}
}

func TestDeduplicate_DifferentIntermediates(t *testing.T) {
	t.Parallel()

	// Two resources with the same GoName but different API paths due to
	// different intermediate plain nodes. Should be disambiguated.
	resources := []Resource{
		{
			GoName:   "VRFNameAggregateAddress",
			FileName: "resource_gen_vrf_name_aggregate_address.go",
			Kind:     TagNodeResource,
			TagFields: []TagField{
				{PathPrefix: []string{"vrf", "name"}},
				{PathPrefix: []string{"protocols", "bgp", "address-family", "ipv4-unicast", "aggregate-address"}},
			},
			intermediatePath:  []string{"protocols", "bgp", "address-family", "ipv4-unicast"},
			namingPath:        []string{"vrf", "name", "aggregate-address"},
			namingIsContainer: []bool{true, false, false},
		},
		{
			GoName:   "VRFNameAggregateAddress",
			FileName: "resource_gen_vrf_name_aggregate_address.go",
			Kind:     TagNodeResource,
			TagFields: []TagField{
				{PathPrefix: []string{"vrf", "name"}},
				{PathPrefix: []string{"protocols", "bgp", "address-family", "ipv6-unicast", "aggregate-address"}},
			},
			intermediatePath:  []string{"protocols", "bgp", "address-family", "ipv6-unicast"},
			namingPath:        []string{"vrf", "name", "aggregate-address"},
			namingIsContainer: []bool{true, false, false},
		},
	}

	got := Deduplicate(resources)
	if len(got) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(got))
	}

	names := map[string]bool{}
	for _, r := range got {
		names[r.GoName] = true
	}

	// Adding 1 trailing intermediate: "ipv4-unicast" vs "ipv6-unicast" should suffice.
	if !names["VRFNameIPv4UnicastAggregateAddress"] {
		t.Errorf("missing VRFNameIPv4UnicastAggregateAddress, got %v", got[0].GoName)
	}
	if !names["VRFNameIPv6UnicastAggregateAddress"] {
		t.Errorf("missing VRFNameIPv6UnicastAggregateAddress, got %v", got[1].GoName)
	}

	// FileNames should also be updated.
	for _, r := range got {
		if r.GoName == "VRFNameIPv4UnicastAggregateAddress" {
			want := "resource_gen_vrf_name_ipv4_unicast_aggregate_address.go"
			if r.FileName != want {
				t.Errorf("FileName = %q, want %q", r.FileName, want)
			}
		}
	}
}

func TestDeduplicate_NeedsMultipleSegments(t *testing.T) {
	t.Parallel()

	// Two resources where the last intermediate segment is the same,
	// requiring 2 trailing segments for disambiguation.
	resources := []Resource{
		{
			GoName:   "VRFNameNetwork",
			FileName: "resource_gen_vrf_name_network.go",
			Kind:     TagNodeResource,
			TagFields: []TagField{
				{PathPrefix: []string{"vrf", "name"}},
				{PathPrefix: []string{"protocols", "bgp", "ipv4-unicast", "network"}},
			},
			intermediatePath:  []string{"protocols", "bgp", "ipv4-unicast"},
			namingPath:        []string{"vrf", "name", "network"},
			namingIsContainer: []bool{true, false, false},
		},
		{
			GoName:   "VRFNameNetwork",
			FileName: "resource_gen_vrf_name_network.go",
			Kind:     TagNodeResource,
			TagFields: []TagField{
				{PathPrefix: []string{"vrf", "name"}},
				{PathPrefix: []string{"protocols", "ospf", "ipv4-unicast", "network"}},
			},
			intermediatePath:  []string{"protocols", "ospf", "ipv4-unicast"},
			namingPath:        []string{"vrf", "name", "network"},
			namingIsContainer: []bool{true, false, false},
		},
	}

	got := Deduplicate(resources)
	if len(got) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(got))
	}

	names := map[string]bool{}
	for _, r := range got {
		names[r.GoName] = true
	}

	// 1 segment ("ipv4-unicast") is the same for both, so need 2 segments.
	if !names["VRFNameBGPIPv4UnicastNetwork"] {
		t.Errorf("missing VRFNameBGPIPv4UnicastNetwork in %v", names)
	}
	if !names["VRFNameOSPFIPv4UnicastNetwork"] {
		t.Errorf("missing VRFNameOSPFIPv4UnicastNetwork in %v", names)
	}
}
