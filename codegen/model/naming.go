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

// Deduplicate resolves duplicate GoName collisions in a resource list.
// Two types of duplicates are handled:
//  1. Same API path (duplicate XML definitions): keep the resource with the
//     most fields, drop the rest.
//  2. Different API paths (lost intermediate segments): disambiguate by
//     inserting intermediate path segments into GoName and FileName.
func Deduplicate(resources []Resource) []Resource {
	// Group resources by GoName, preserving first-seen order.
	type group struct {
		indices []int
	}
	groups := make(map[string]*group)
	var order []string

	for i, r := range resources {
		g, ok := groups[r.GoName]
		if !ok {
			g = &group{}
			groups[r.GoName] = g
			order = append(order, r.GoName)
		}
		g.indices = append(g.indices, i)
	}

	result := make([]Resource, 0, len(resources))

	for _, name := range order {
		g := groups[name]
		if len(g.indices) == 1 {
			result = append(result, resources[g.indices[0]])
			continue
		}

		// Check if all resources in the group have the same API path.
		allSamePath := true
		refPath := strings.Join(resources[g.indices[0]].BasePath(), "/")
		for _, idx := range g.indices[1:] {
			if strings.Join(resources[idx].BasePath(), "/") != refPath {
				allSamePath = false
				break
			}
		}

		if allSamePath {
			// Duplicate XML definitions: keep the one with the most fields.
			bestIdx := g.indices[0]
			for _, idx := range g.indices[1:] {
				if len(resources[idx].Fields) > len(resources[bestIdx].Fields) {
					bestIdx = idx
				}
			}
			result = append(result, resources[bestIdx])
			continue
		}

		// Different API paths: disambiguate by adding intermediate segments.
		result = append(result, disambiguateGroup(resources, g.indices)...)
	}

	return result
}

// disambiguateGroup renames resources in a collision group by inserting
// trailing intermediate path segments until all GoNames are unique.
func disambiguateGroup(resources []Resource, indices []int) []Resource {
	// Compute effective intermediates for each resource in the group.
	// This captures both the resource's own intermediates and any inherited
	// from ancestor resources.
	infos := make([]interInfo, len(indices))
	maxLen := 0

	for i, idx := range indices {
		segs, insertIdx := effectiveIntermediates(resources[idx])
		infos[i] = interInfo{segs, insertIdx}
		if len(segs) > maxLen {
			maxLen = len(segs)
		}
	}

	// Try adding N trailing intermediate segments until all names are unique.
	for n := 1; n <= maxLen; n++ {
		names := make(map[string]bool)
		unique := true

		for i, idx := range indices {
			np, nc := disambiguatedNamingPath(resources[idx], infos[i].segments, infos[i].insertIdx, n)
			name := BuildGoResourceName(np, nc)
			if names[name] {
				unique = false
				break
			}
			names[name] = true
		}

		if unique {
			return applyDisambiguation(resources, indices, infos, n)
		}
	}

	// Fallback: use all intermediate segments.
	return applyDisambiguation(resources, indices, infos, maxLen)
}

// interInfo holds computed intermediate information for disambiguation.
type interInfo struct {
	segments  []string
	insertIdx int
}

// applyDisambiguation rebuilds GoName and FileName for each resource in the
// group using n trailing intermediate segments.
func applyDisambiguation(resources []Resource, indices []int, infos []interInfo, n int) []Resource {
	out := make([]Resource, len(indices))
	for i, idx := range indices {
		r := resources[idx]
		np, nc := disambiguatedNamingPath(r, infos[i].segments, infos[i].insertIdx, n)
		r.GoName = BuildGoResourceName(np, nc)
		r.FileName = BuildFileName(np, nc)
		r.namingPath = np
		r.namingIsContainer = nc
		out[i] = r
	}
	return out
}

// effectiveIntermediates computes the intermediate plain node segments for a
// resource by comparing its API path (BasePath) with its naming path. This
// captures both the resource's own intermediates and intermediates inherited
// from ancestor resources. Also returns the insertion index in the naming
// path where disambiguation segments should be inserted.
func effectiveIntermediates(r Resource) ([]string, int) {
	bp := r.BasePath()
	np := r.namingPath
	if len(bp) == 0 || len(np) == 0 {
		return nil, 0
	}

	var allInter []string
	insertIdx := len(np) - 1 // default: before last segment
	foundFirst := false
	bpIdx := 0

	for npIdx := 0; npIdx < len(np) && bpIdx < len(bp); npIdx++ {
		for bpIdx < len(bp) && bp[bpIdx] != np[npIdx] {
			allInter = append(allInter, bp[bpIdx])
			if !foundFirst {
				insertIdx = npIdx
				foundFirst = true
			}
			bpIdx++
		}
		if bpIdx < len(bp) {
			bpIdx++
		}
	}

	return allInter, insertIdx
}

// disambiguatedNamingPath inserts n trailing segments from the given
// intermediates into the naming path at insertIdx. Returns the expanded
// path and isContainer slices.
func disambiguatedNamingPath(r Resource, inter []string, insertIdx int, n int) ([]string, []bool) {
	np := r.namingPath
	nc := r.namingIsContainer

	if len(inter) == 0 || len(np) == 0 {
		return np, nc
	}

	start := len(inter) - n
	if start < 0 {
		start = 0
	}
	segments := inter[start:]

	newPath := make([]string, 0, len(np)+len(segments))
	newIsContainer := make([]bool, 0, len(np)+len(segments))

	newPath = append(newPath, np[:insertIdx]...)
	newIsContainer = append(newIsContainer, nc[:insertIdx]...)

	// Disambiguation segments are plain nodes, treat as containers.
	for _, seg := range segments {
		newPath = append(newPath, seg)
		newIsContainer = append(newIsContainer, true)
	}

	newPath = append(newPath, np[insertIdx:]...)
	newIsContainer = append(newIsContainer, nc[insertIdx:]...)

	return newPath, newIsContainer
}

func titleCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
