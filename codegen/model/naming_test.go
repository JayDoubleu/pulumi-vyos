package model

import "testing"

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
