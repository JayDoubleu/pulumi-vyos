// Package generate loads Go templates and emits formatted resource files from
// the code-generation intermediate representation.
package generate

import (
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/jaydoubleu/pulumi-vyos/codegen/model"
)

//go:embed templates/*.go.tmpl
var templateFS embed.FS

// Generate writes all generated resource files to outputDir.
func Generate(resources []model.Resource, outputDir string) error {
	funcMap := template.FuncMap{
		"vyosPathLit":    vyosPathLit,
		"isOptionalType": isOptionalType,
		"escapeGo":       escapeGo,
		"checkBody":      checkBody,
	}

	resourceTmpl, err := template.New("resource.go.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/resource.go.tmpl")
	if err != nil {
		return fmt.Errorf("parse resource template: %w", err)
	}

	regTmpl, err := template.New("registration.go.tmpl").ParseFS(templateFS, "templates/registration.go.tmpl")
	if err != nil {
		return fmt.Errorf("parse registration template: %w", err)
	}

	// Write helpers file (static content, not a template).
	if err := writeStaticFile("templates/helpers.go.tmpl", filepath.Join(outputDir, "resource_gen_helpers.go")); err != nil {
		return fmt.Errorf("write helpers: %w", err)
	}

	// Write per-resource files.
	for i := range resources {
		td := NewTemplateData(&resources[i])
		outPath := filepath.Join(outputDir, resources[i].FileName)
		if err := writeTemplate(resourceTmpl, td, outPath); err != nil {
			return fmt.Errorf("write %s: %w", resources[i].FileName, err)
		}
	}

	// Write registration file.
	type regData struct {
		Resources []model.Resource
	}
	regPath := filepath.Join(outputDir, "resource_gen_registration.go")
	if err := writeTemplate(regTmpl, regData{Resources: resources}, regPath); err != nil {
		return fmt.Errorf("write registration: %w", err)
	}

	return nil
}

// writeStaticFile reads a file from the embedded FS and writes it with gofmt.
// Used for template files that contain Go source but no template directives
// (e.g., the helpers file has {{}} composite literals that conflict with
// template syntax).
func writeStaticFile(embeddedPath, outPath string) error {
	raw, err := templateFS.ReadFile(embeddedPath)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", embeddedPath, err)
	}

	formatted, err := format.Source(raw)
	if err != nil {
		return fmt.Errorf("gofmt %s: %w", filepath.Base(outPath), err)
	}

	return os.WriteFile(outPath, formatted, 0o644)
}

// writeTemplate executes a template and writes gofmt'd output to a file.
func writeTemplate(tmpl *template.Template, data any, path string) error {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		return fmt.Errorf("gofmt %s: %w\nraw output:\n%s", filepath.Base(path), err, buf.String())
	}

	return os.WriteFile(path, formatted, 0o644)
}

// vyosPathLit converts a string slice to a Go literal list: "a", "b", "c".
func vyosPathLit(path []string) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = fmt.Sprintf("%q", p)
	}
	return strings.Join(parts, ", ")
}

// isOptionalType returns true if the Go type represents an optional field
// (pointer or slice), meaning it should get the ,optional Pulumi tag.
func isOptionalType(goType string) bool {
	return strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]")
}

// escapeGo escapes a string for safe inclusion inside a Go string literal.
// It handles quotes, backslashes, newlines, and other special characters.
func escapeGo(s string) string {
	q := strconv.Quote(s)
	return q[1 : len(q)-1]
}

// TemplateData wraps a Resource with computed properties for template rendering.
type TemplateData struct {
	*model.Resource
}

// NewTemplateData creates a TemplateData from a Resource.
func NewTemplateData(r *model.Resource) TemplateData {
	return TemplateData{Resource: r}
}

// IsTagNode returns true if this is a TagNodeResource.
func (td TemplateData) IsTagNode() bool {
	return td.Kind == model.TagNodeResource
}

// LowerName returns the GoName with the leading character(s) lowercased,
// used as a prefix for the basePath function name.
func (td TemplateData) LowerName() string {
	return model.PascalToCamel(td.GoName)
}

// BasePathParams returns the Go function parameter list for the basePath
// function: "name string" or "zoneName, name string".
func (td TemplateData) BasePathParams() string {
	names := make([]string, len(td.TagFields))
	for i, tf := range td.TagFields {
		names[i] = tf.ParamName()
	}
	return strings.Join(names, ", ") + " string"
}

// BasePathElements returns the Go expression for building the base path []any
// literal, interleaving quoted static segments with unquoted parameter names.
func (td TemplateData) BasePathElements() string {
	var parts []string
	for _, tf := range td.TagFields {
		for _, seg := range tf.PathPrefix {
			parts = append(parts, fmt.Sprintf("%q", seg))
		}
		parts = append(parts, tf.ParamName())
	}
	return strings.Join(parts, ", ")
}

// ReadBasePathArgs returns arguments for calling basePath in Read.
func (td TemplateData) ReadBasePathArgs() string {
	return td.tagFieldArgs("req.State.")
}

// ParseConfigCallArgs returns arguments for calling parseConfig in Read.
func (td TemplateData) ParseConfigCallArgs() string {
	return td.tagFieldArgs("req.State.")
}

// ParseConfigParams returns the function parameter list for parseConfig.
func (td TemplateData) ParseConfigParams() string {
	return td.BasePathParams()
}

// BuildOpsBasePathArgs returns arguments for calling basePath in buildOps.
func (td TemplateData) BuildOpsBasePathArgs() string {
	return td.tagFieldArgs("args.")
}

// UpdateOpsBasePathArgs returns arguments for calling basePath in buildUpdateOps.
func (td TemplateData) UpdateOpsBasePathArgs() string {
	return td.tagFieldArgs("cur.")
}

// DeleteBasePathArgs returns arguments for calling basePath in Delete.
func (td TemplateData) DeleteBasePathArgs() string {
	return td.tagFieldArgs("req.State.")
}

// tagFieldArgs joins tag field GoNames with a prefix and comma separator.
func (td TemplateData) tagFieldArgs(prefix string) string {
	names := make([]string, len(td.TagFields))
	for i, tf := range td.TagFields {
		names[i] = prefix + tf.GoName
	}
	return strings.Join(names, ", ")
}

// ReadEmptyArgs returns Go code that constructs a minimal Args struct with only
// tag fields populated from req.State. Used when ShowConfig returns "empty"
// for a tag node that exists but has no child properties.
func (td TemplateData) ReadEmptyArgs() string {
	var parts []string
	for _, tf := range td.TagFields {
		parts = append(parts, fmt.Sprintf("%s: req.State.%s", tf.GoName, tf.GoName))
	}
	return td.GoName + "Args{" + strings.Join(parts, ", ") + "}"
}

// LeafFieldGoName returns the Go field name for a leaf resource's value field.
func (td TemplateData) LeafFieldGoName() string {
	if len(td.Fields) > 0 {
		return td.Fields[0].GoName
	}
	return ""
}

// LeafPathLit returns the Go string slice literal for the full leaf path.
func (td TemplateData) LeafPathLit() string {
	return "[]string{" + vyosPathLit(td.LeafPath) + "}"
}

// LeafParentPathLit returns the Go string slice literal for the leaf parent path
// (everything except the last segment).
func (td TemplateData) LeafParentPathLit() string {
	if len(td.LeafPath) < 2 {
		return "[]string{}"
	}
	return "[]string{" + vyosPathLit(td.LeafPath[:len(td.LeafPath)-1]) + "}"
}

// LeafVyosName returns the last segment of the leaf path.
func (td TemplateData) LeafVyosName() string {
	if len(td.LeafPath) > 0 {
		return td.LeafPath[len(td.LeafPath)-1]
	}
	return ""
}

// NeedsRegexp returns true if any constraint has regex patterns.
func (td TemplateData) NeedsRegexp() bool {
	for _, tf := range td.TagFields {
		if tf.Constraint != nil && len(tf.Constraint.Patterns) > 0 {
			return true
		}
	}
	for _, f := range td.Fields {
		if f.Constraint != nil && len(f.Constraint.Patterns) > 0 {
			return true
		}
	}
	return false
}

// RegexVarInfo holds data for a compiled regex variable declaration.
type RegexVarInfo struct {
	VarName    string
	PatternLit string
}

// RegexVars returns the list of regex variable declarations for this resource.
func (td TemplateData) RegexVars() []RegexVarInfo {
	lower := td.LowerName()
	var vars []RegexVarInfo
	for _, tf := range td.TagFields {
		if tf.Constraint != nil && len(tf.Constraint.Patterns) > 0 {
			vars = append(vars, RegexVarInfo{
				VarName:    lower + tf.GoName + "Re",
				PatternLit: strconv.Quote(strings.Join(tf.Constraint.Patterns, "|")),
			})
		}
	}
	for _, f := range td.Fields {
		if f.Constraint != nil && len(f.Constraint.Patterns) > 0 {
			vars = append(vars, RegexVarInfo{
				VarName:    lower + f.GoName + "Re",
				PatternLit: strconv.Quote(strings.Join(f.Constraint.Patterns, "|")),
			})
		}
	}
	return vars
}

// checkBody generates the Go validation code for a Check method body.
func checkBody(td TemplateData) string {
	var b strings.Builder
	lower := td.LowerName()

	for _, tf := range td.TagFields {
		if tf.Constraint == nil {
			continue
		}
		c := tf.Constraint
		if len(c.Patterns) > 0 {
			varName := lower + tf.GoName + "Re"
			errMsg := strconv.Quote(c.ErrorMessage)
			fmt.Fprintf(&b, "\tif f := checkRegex(inputs.%s, %s, %q, %s); f != nil {\n",
				tf.GoName, varName, tf.PulumiName, errMsg)
			b.WriteString("\t\tfailures = append(failures, *f)\n")
			b.WriteString("\t}\n")
		}
	}

	for _, f := range td.Fields {
		if f.Constraint == nil {
			continue
		}
		c := f.Constraint
		errMsg := strconv.Quote(c.ErrorMessage)

		if len(c.Patterns) > 0 {
			varName := lower + f.GoName + "Re"
			switch f.FieldType {
			case model.StringField:
				if strings.HasPrefix(f.GoType, "*") {
					fmt.Fprintf(&b, "\tif inputs.%s != nil {\n", f.GoName)
					fmt.Fprintf(&b, "\t\tif f := checkRegex(*inputs.%s, %s, %q, %s); f != nil {\n",
						f.GoName, varName, f.PulumiName, errMsg)
					b.WriteString("\t\t\tfailures = append(failures, *f)\n")
					b.WriteString("\t\t}\n")
					b.WriteString("\t}\n")
				} else {
					fmt.Fprintf(&b, "\tif f := checkRegex(inputs.%s, %s, %q, %s); f != nil {\n",
						f.GoName, varName, f.PulumiName, errMsg)
					b.WriteString("\t\tfailures = append(failures, *f)\n")
					b.WriteString("\t}\n")
				}
			case model.MultiField:
				fmt.Fprintf(&b, "\tfor _, v := range inputs.%s {\n", f.GoName)
				fmt.Fprintf(&b, "\t\tif f := checkRegex(v, %s, %q, %s); f != nil {\n",
					varName, f.PulumiName, errMsg)
				b.WriteString("\t\t\tfailures = append(failures, *f)\n")
				b.WriteString("\t\t\tbreak\n")
				b.WriteString("\t\t}\n")
				b.WriteString("\t}\n")
			}
		}

		if c.NumericRange != nil && f.FieldType == model.IntField {
			if strings.HasPrefix(f.GoType, "*") {
				fmt.Fprintf(&b, "\tif inputs.%s != nil {\n", f.GoName)
				fmt.Fprintf(&b, "\t\tif f := checkIntRange(int64(*inputs.%s), %d, %d, %q, %s); f != nil {\n",
					f.GoName, c.NumericRange.Min, c.NumericRange.Max, f.PulumiName, errMsg)
				b.WriteString("\t\t\tfailures = append(failures, *f)\n")
				b.WriteString("\t\t}\n")
				b.WriteString("\t}\n")
			} else {
				fmt.Fprintf(&b, "\tif f := checkIntRange(int64(inputs.%s), %d, %d, %q, %s); f != nil {\n",
					f.GoName, c.NumericRange.Min, c.NumericRange.Max, f.PulumiName, errMsg)
				b.WriteString("\t\tfailures = append(failures, *f)\n")
				b.WriteString("\t}\n")
			}
		}
	}

	return b.String()
}
