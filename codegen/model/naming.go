package model

import (
	"strings"
	"unicode"
)

// acronyms maps lowercase VyOS tokens to their Go-idiomatic uppercase form.
var acronyms = map[string]string{
	"mtu":   "MTU",
	"id":    "ID",
	"ip":    "IP",
	"ipv4":  "IPv4",
	"ipv6":  "IPv6",
	"mac":   "MAC",
	"dns":   "DNS",
	"ssh":   "SSH",
	"tcp":   "TCP",
	"udp":   "UDP",
	"tls":   "TLS",
	"ssl":   "SSL",
	"vlan":  "VLAN",
	"vni":   "VNI",
	"vrf":   "VRF",
	"vti":   "VTI",
	"ttl":   "TTL",
	"url":   "URL",
	"uri":   "URI",
	"api":   "API",
	"bgp":   "BGP",
	"ospf":  "OSPF",
	"mpls":  "MPLS",
	"rip":   "RIP",
	"dhcp":  "DHCP",
	"ntp":   "NTP",
	"ppp":   "PPP",
	"pppoe": "PPPoE",
	"gre":   "GRE",
	"vxlan": "VXLAN",
	"nat":   "NAT",
	"acl":   "ACL",
	"cpu":   "CPU",
	"gro":   "GRO",
	"gso":   "GSO",
	"lro":   "LRO",
	"tso":   "TSO",
	"rps":   "RPS",
	"rfs":   "RFS",
	"sg":    "SG",
	"rx":    "RX",
	"tx":    "TX",
	"hw":    "HW",
	"qos":   "QoS",
	"arp":   "ARP",
	"cqe":   "CQE",
}

// reservedPulumiNames are Pulumi property names that conflict with framework
// internals (e.g., "id" is the resource identity). Fields with these names
// get a "Value" suffix: "id" -> "idValue", "ID" -> "IDValue".
var reservedPulumiNames = map[string]bool{
	"id": true,
}

// singulars maps plural container names to singular forms.
var singulars = map[string]string{
	"interfaces": "interface",
	"protocols":  "protocol",
	"services":   "service",
}

// KebabToPascal converts a VyOS kebab-case name to Go PascalCase.
// Examples: "host-name" -> "HostName", "mtu" -> "MTU", "ipv6-address" -> "IPv6Address".
// Names starting with a digit get a "N" prefix: "6rd-prefix" -> "N6RDPrefix".
func KebabToPascal(kebab string) string {
	parts := strings.Split(kebab, "-")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if acronym, ok := acronyms[strings.ToLower(part)]; ok {
			b.WriteString(acronym)
		} else {
			b.WriteString(titleCase(part))
		}
	}
	result := b.String()
	// Go identifiers cannot start with a digit.
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "N" + result
	}
	return result
}

// PascalToCamel converts a Go PascalCase name to camelCase for Pulumi tags.
// Rules:
//   - Full-acronym names lowercase entirely: "MTU" -> "mtu"
//   - Leading acronym run lowercased except last char: "IPAddress" -> "ipAddress"
//   - Single leading uppercase: "Description" -> "description"
func PascalToCamel(pascal string) string {
	if len(pascal) == 0 {
		return ""
	}

	runes := []rune(pascal)

	// Find the length of the leading uppercase run.
	upperRun := 0
	for upperRun < len(runes) && unicode.IsUpper(runes[upperRun]) {
		upperRun++
	}

	if upperRun == 0 {
		return pascal
	}

	// Entire name is uppercase (e.g., "MTU", "MAC").
	if upperRun == len(runes) {
		return strings.ToLower(pascal)
	}

	result := make([]rune, len(runes))
	copy(result, runes)

	if upperRun == 1 {
		// Single leading uppercase: just lowercase the first char.
		result[0] = unicode.ToLower(result[0])
	} else {
		// Multiple leading uppercase chars (e.g., "IPAddress", "IPv6Name").
		// Lowercase all except the last char of the run (which starts the next word).
		for i := 0; i < upperRun-1; i++ {
			result[i] = unicode.ToLower(result[i])
		}
	}

	return string(result)
}

// SingularizePath applies singularization to a container node name.
func SingularizePath(name string) string {
	if s, ok := singulars[name]; ok {
		return s
	}
	return name
}

// BuildGoResourceName builds the PascalCase Go name for a resource from its
// full path segments. Container nodes (regular nodes) are singularized.
func BuildGoResourceName(pathSegments []string, isContainer []bool) string {
	var b strings.Builder
	for i, seg := range pathSegments {
		name := seg
		if i < len(isContainer) && isContainer[i] {
			name = SingularizePath(name)
		}
		b.WriteString(KebabToPascal(name))
	}
	return b.String()
}

// BuildGoFieldName builds the PascalCase Go name for a field, flattening
// any prefix path segments (from nested plain nodes).
func BuildGoFieldName(prefix []string, leafName string) string {
	var b strings.Builder
	for _, p := range prefix {
		b.WriteString(KebabToPascal(p))
	}
	b.WriteString(KebabToPascal(leafName))
	return b.String()
}

// BuildFileName creates the generated file name for a resource.
// Example: ["interfaces", "ethernet"] -> "resource_gen_interface_ethernet.go"
// Avoids producing names ending in _test.go (which Go treats as test files).
func BuildFileName(pathSegments []string, isContainer []bool) string {
	parts := make([]string, len(pathSegments))
	for i, seg := range pathSegments {
		name := seg
		if i < len(isContainer) && isContainer[i] {
			name = SingularizePath(name)
		}
		parts[i] = strings.ReplaceAll(name, "-", "_")
	}
	name := "resource_gen_" + strings.Join(parts, "_") + ".go"
	// Go treats files ending in _test.go as test files, so rename to avoid that.
	if strings.HasSuffix(name, "_test.go") {
		name = name[:len(name)-len("_test.go")] + "_res.go"
	}
	return name
}

// FixReservedFieldName adjusts a Go name and Pulumi name pair if the Pulumi
// name would collide with a Pulumi reserved name. Returns the (possibly
// adjusted) Go and Pulumi names.
func FixReservedFieldName(goName, pulumiName string) (string, string) {
	if reservedPulumiNames[pulumiName] {
		return goName + "Value", pulumiName + "Value"
	}
	return goName, pulumiName
}

func titleCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
