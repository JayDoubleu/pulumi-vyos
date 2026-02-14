// Package model defines the intermediate representation (IR) used by the code
// generator to emit Go resource files from parsed VyOS XML definitions.
package model

// Resource represents a single Pulumi resource derived from a VyOS XML definition.
type Resource struct {
	GoName      string       // PascalCase: "InterfaceEthernet"
	Description string       // From XML <help>
	Kind        ResourceKind // TagNodeResource or LeafNodeResource
	Fields      []Field
	FileName    string // "resource_gen_interface_ethernet.go"

	// TagFields describes the tag node key fields for this resource, in path
	// order from outermost to innermost. For a simple (non-nested) TagNodeResource,
	// there is exactly one entry. For LeafNodeResource, this is empty.
	TagFields []TagField

	// LeafPath is the complete static VyOS config path for a LeafNodeResource
	// (e.g., ["system", "host-name"]). Empty for TagNodeResource.
	LeafPath []string

	// Unexported fields for deduplication (set by builder, used by Deduplicate).
	intermediatePath  []string // plain nodes between parent tag and this tag
	namingPath        []string // path segments used for GoName
	namingIsContainer []bool   // isContainer flags for naming path
}

// ResourceKind distinguishes tag-node resources from leaf-node resources.
type ResourceKind int

const (
	// TagNodeResource has a Name field and uses BatchConfigure.
	TagNodeResource ResourceKind = iota
	// LeafNodeResource is a single-value resource using Set/Delete.
	LeafNodeResource
)

// TagField describes one tag node key segment in the resource's VyOS path.
type TagField struct {
	GoName      string           // "Name", "ZoneName"
	PulumiName  string           // "name", "zoneName"
	Description string           // From the tagNode's <help>
	PathPrefix  []string         // Static path segments before this tag value
	Constraint  *FieldConstraint // Validation constraint from XML
}

// Field represents one configurable property on a resource.
type Field struct {
	GoName      string           // PascalCase: "DisableFlowControl"
	PulumiName  string           // camelCase: "disableFlowControl"
	VyosName    string           // Original last segment: "disable-flow-control"
	VyosPath    []string         // Relative path from resource base: ["offload", "gro"]
	GoType      string           // "*string", "*int", "*bool", "[]string"
	FieldType   FieldType        // How CRUD handles this field
	Description string           // From XML <help>
	Constraint  *FieldConstraint // Validation constraint from XML
}

// FieldConstraint holds parsed validation rules for a field.
type FieldConstraint struct {
	Patterns     []string // Regex patterns from <regex>
	NumericRange *Range   // Parsed from validator name="numeric" argument="--range N-M"
	ErrorMessage string   // From <constraintErrorMessage>
}

// Range represents a numeric min/max constraint.
type Range struct {
	Min int64
	Max int64
}

// FieldType controls how the code generator emits CRUD operations for a field.
type FieldType int

// FieldType constants for each supported VyOS field kind.
const (
	StringField FieldType = iota // *string, set with value
	IntField                     // *int, set as string, parse from string/number
	BoolField                    // *bool, valueless (set path only, no value)
	MultiField                   // []string, each value in path (no value field)
)

// HasConstraints returns true if any field or tag field has validation constraints.
func (r *Resource) HasConstraints() bool {
	for _, tf := range r.TagFields {
		if tf.Constraint != nil {
			return true
		}
	}
	for _, f := range r.Fields {
		if f.Constraint != nil {
			return true
		}
	}
	return false
}

// NeedsStrconv returns true if any field requires the strconv package.
func (r *Resource) NeedsStrconv() bool {
	for _, f := range r.Fields {
		if f.FieldType == IntField {
			return true
		}
	}
	return false
}

// ParamName returns the camelCase parameter name for this tag field.
func (tf TagField) ParamName() string {
	return PascalToCamel(tf.GoName)
}

// BasePath returns the full static VyOS path for building operations.
// For TagNodeResource this concatenates all TagField PathPrefixes.
// For LeafNodeResource this returns LeafPath.
func (r *Resource) BasePath() []string {
	if r.Kind == LeafNodeResource {
		return r.LeafPath
	}
	var path []string
	for _, tf := range r.TagFields {
		path = append(path, tf.PathPrefix...)
	}
	return path
}
