package model

import (
	"strings"

	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

// InferFieldType determines the FieldType from an XML leafNode's properties.
// Rules (from DESIGN.md):
//  1. <valueless/> present -> BoolField
//  2. All <valueHelp> formats match u32:* -> IntField
//  3. <multi/> present -> MultiField
//  4. Default -> StringField
func InferFieldType(props *xmlparse.Properties) FieldType {
	if props == nil {
		return StringField
	}

	if props.Valueless != nil {
		return BoolField
	}

	if allU32ValueHelps(props.ValueHelps) {
		return IntField
	}

	if props.Multi != nil {
		return MultiField
	}

	return StringField
}

// GoTypeFor returns the Go type string for a given FieldType.
func GoTypeFor(ft FieldType) string {
	switch ft {
	case BoolField:
		return "*bool"
	case IntField:
		return "*int"
	case MultiField:
		return "[]string"
	default:
		return "*string"
	}
}

// GoTypeForRequired returns the Go type string for a required (non-optional)
// field, such as the value field of a LeafNodeResource. Unlike GoTypeFor, these
// are plain types (string, int, bool) rather than pointer types.
func GoTypeForRequired(ft FieldType) string {
	switch ft {
	case BoolField:
		return "bool"
	case IntField:
		return "int"
	case MultiField:
		return "[]string"
	default:
		return "string"
	}
}

// allU32ValueHelps returns true if there is at least one valueHelp and every
// valueHelp format starts with "u32:".
func allU32ValueHelps(helps []xmlparse.ValueHelp) bool {
	if len(helps) == 0 {
		return false
	}
	for _, h := range helps {
		if !strings.HasPrefix(h.Format, "u32:") && !strings.HasPrefix(h.Format, "u32") {
			return false
		}
	}
	return true
}
