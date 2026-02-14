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
