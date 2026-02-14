package model

import (
	"strings"

	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

// Build converts a parsed VyOS XML interface definition into a list of
// Pulumi resources. It walks the XML tree, detects resource boundaries
// (tagNodes, leafNodes with owners), and flattens nested plain nodes
// into field prefixes.
func Build(def *xmlparse.InterfaceDefinition) []Resource {
	var resources []Resource

	for i := range def.Nodes {
		resources = append(resources, walkNode(&def.Nodes[i], nil, nil)...)
	}
	for i := range def.TagNodes {
		resources = append(resources, buildTagNodeResource(&def.TagNodes[i], nil, nil)...)
	}
	for i := range def.LeafNodes {
		if def.LeafNodes[i].Owner != "" {
			if r, ok := buildLeafNodeResource(&def.LeafNodes[i], nil, nil); ok {
				resources = append(resources, r)
			}
		}
	}

	return resources
}

// walkNode traverses a plain node, looking for resource boundaries in its children.
// pathSegments accumulates the VyOS path from the root. isContainer tracks
// whether each segment is a plain container node (for singularization).
func walkNode(n *xmlparse.Node, pathSegments []string, isContainer []bool) []Resource {
	newPath := append(append([]string{}, pathSegments...), n.Name)
	newIsContainer := append(append([]bool{}, isContainer...), true)

	if n.Children == nil {
		return nil
	}

	var resources []Resource

	for i := range n.Children.TagNodes {
		resources = append(resources,
			buildTagNodeResource(&n.Children.TagNodes[i], newPath, newIsContainer)...)
	}

	for i := range n.Children.LeafNodes {
		if n.Children.LeafNodes[i].Owner != "" {
			if r, ok := buildLeafNodeResource(&n.Children.LeafNodes[i], newPath, newIsContainer); ok {
				resources = append(resources, r)
			}
		}
	}

	for i := range n.Children.Nodes {
		cn := &n.Children.Nodes[i]
		if cn.Owner != "" {
			// Owned node: its children may contain resources.
			resources = append(resources, walkOwnedNode(cn, newPath, newIsContainer)...)
		} else {
			resources = append(resources, walkNode(cn, newPath, newIsContainer)...)
		}
	}

	return resources
}

// walkOwnedNode handles a node that has an owner attribute. Its tagNode
// children become resources. We also recurse into plain node children
// to find deeper tagNodes.
func walkOwnedNode(n *xmlparse.Node, parentPath []string, parentIsContainer []bool) []Resource {
	newPath := append(append([]string{}, parentPath...), n.Name)
	// Owned nodes are not pure containers (they represent a config script boundary),
	// but for naming we still treat them as containers since they are node elements.
	newIsContainer := append(append([]bool{}, parentIsContainer...), false)

	if n.Children == nil {
		return nil
	}

	var resources []Resource

	for i := range n.Children.TagNodes {
		resources = append(resources,
			buildTagNodeResource(&n.Children.TagNodes[i], newPath, newIsContainer)...)
	}

	for i := range n.Children.LeafNodes {
		if n.Children.LeafNodes[i].Owner != "" {
			if r, ok := buildLeafNodeResource(&n.Children.LeafNodes[i], newPath, newIsContainer); ok {
				resources = append(resources, r)
			}
		}
	}

	// Recurse into plain node children (e.g., firewall > group > address-group).
	for i := range n.Children.Nodes {
		cn := &n.Children.Nodes[i]
		if cn.Owner != "" {
			resources = append(resources, walkOwnedNode(cn, newPath, newIsContainer)...)
		} else {
			resources = append(resources, walkGroupNode(cn, newPath, newIsContainer)...)
		}
	}

	return resources
}

// walkGroupNode handles a plain node inside an owned context (e.g., firewall > group).
// It looks for tagNode children that should become resources.
func walkGroupNode(n *xmlparse.Node, parentPath []string, parentIsContainer []bool) []Resource {
	newPath := append(append([]string{}, parentPath...), n.Name)
	newIsContainer := append(append([]bool{}, parentIsContainer...), true)

	if n.Children == nil {
		return nil
	}

	var resources []Resource

	for i := range n.Children.TagNodes {
		resources = append(resources,
			buildTagNodeResource(&n.Children.TagNodes[i], newPath, newIsContainer)...)
	}

	// Recurse into deeper plain nodes.
	for i := range n.Children.Nodes {
		resources = append(resources,
			walkGroupNode(&n.Children.Nodes[i], newPath, newIsContainer)...)
	}

	return resources
}

// buildTagNodeResource creates a Resource from a tagNode and recursively
// handles nested tagNodes as sub-resources.
func buildTagNodeResource(tn *xmlparse.TagNode, parentPath []string, parentIsContainer []bool) []Resource {
	resourcePath := append(append([]string{}, parentPath...), tn.Name)
	resourceIsContainer := append(append([]bool{}, parentIsContainer...), false)

	goName := BuildGoResourceName(resourcePath, resourceIsContainer)
	fileName := BuildFileName(resourcePath, resourceIsContainer)

	helpText := ""
	if tn.Properties != nil {
		helpText = tn.Properties.Help
	}

	res := Resource{
		GoName:      goName,
		Description: helpText,
		Kind:        TagNodeResource,
		FileName:    fileName,
		TagFields: []TagField{
			{
				GoName:      "Name",
				PulumiName:  "name",
				Description: helpText,
				PathPrefix:  append([]string{}, resourcePath...),
			},
		},
	}

	var subResources []Resource
	if tn.Children != nil {
		res.Fields, subResources = collectFields(tn.Children, resourcePath, resourceIsContainer, nil)
	}

	result := []Resource{res}
	result = append(result, subResources...)
	return result
}

// buildLeafNodeResource creates a Resource from a leafNode with owner.
// Returns (resource, true) on success. Returns (zero, false) if the leaf
// type is not supported as a standalone resource (e.g., multi-value or
// valueless boolean leaves).
func buildLeafNodeResource(ln *xmlparse.LeafNode, parentPath []string, parentIsContainer []bool) (Resource, bool) {
	helpText := ""
	ft := StringField
	if ln.Properties != nil {
		helpText = ln.Properties.Help
		ft = InferFieldType(ln.Properties)
	}

	// The leaf node template only supports StringField types. Multi-value
	// and valueless boolean leaves need different CRUD patterns.
	if ft != StringField {
		return Resource{}, false
	}

	resourcePath := append(append([]string{}, parentPath...), ln.Name)
	resourceIsContainer := append(append([]bool{}, parentIsContainer...), false)

	goName := BuildGoResourceName(resourcePath, resourceIsContainer)
	fileName := BuildFileName(resourcePath, resourceIsContainer)

	fieldGoName := KebabToPascal(ln.Name)
	fieldPulumiName := PascalToCamel(fieldGoName)
	fieldGoName, fieldPulumiName = FixReservedFieldName(fieldGoName, fieldPulumiName)

	return Resource{
		GoName:      goName,
		Description: helpText,
		Kind:        LeafNodeResource,
		FileName:    fileName,
		LeafPath:    append([]string{}, resourcePath...),
		Fields: []Field{
			{
				GoName:      fieldGoName,
				PulumiName:  fieldPulumiName,
				VyosName:    ln.Name,
				VyosPath:    []string{ln.Name},
				GoType:      GoTypeForRequired(ft),
				FieldType:   ft,
				Description: helpText,
			},
		},
	}, true
}

// collectFields gathers fields from a tagNode's children. Plain nodes are
// flattened (their children become fields with prefixed names). Nested
// tagNodes become separate sub-resources.
func collectFields(
	children *xmlparse.Children,
	resourcePath []string,
	resourceIsContainer []bool,
	prefix []string,
) ([]Field, []Resource) {
	var fields []Field
	var subResources []Resource
	seen := make(map[string]bool)

	addField := func(f Field) {
		key := strings.Join(f.VyosPath, "/")
		if seen[key] {
			return
		}
		seen[key] = true
		fields = append(fields, f)
	}

	for _, ln := range children.LeafNodes {
		addField(buildField(&ln, prefix))
	}

	for i := range children.Nodes {
		n := &children.Nodes[i]
		if n.Children != nil {
			newPrefix := append(append([]string{}, prefix...), n.Name)
			childFields, childResources := collectFields(n.Children, resourcePath, resourceIsContainer, newPrefix)
			for _, f := range childFields {
				addField(f)
			}
			subResources = append(subResources, childResources...)
		}
	}

	for i := range children.TagNodes {
		tn := &children.TagNodes[i]
		subResources = append(subResources,
			buildNestedTagNodeResource(tn, resourcePath, resourceIsContainer)...)
	}

	return fields, subResources
}

// buildNestedTagNodeResource creates a sub-resource from a tagNode nested
// within another tagNode (e.g., firewall > zone > from).
func buildNestedTagNodeResource(
	tn *xmlparse.TagNode,
	parentResourcePath []string,
	parentIsContainer []bool,
) []Resource {
	resourcePath := append(append([]string{}, parentResourcePath...), tn.Name)
	resourceIsContainer := append(append([]bool{}, parentIsContainer...), false)

	goName := BuildGoResourceName(resourcePath, resourceIsContainer)
	fileName := BuildFileName(resourcePath, resourceIsContainer)

	helpText := ""
	if tn.Properties != nil {
		helpText = tn.Properties.Help
	}

	// Build TagFields: parent's tag field + this tag's field.
	// The parent resource path includes the parent tag node name.
	// For the nested resource, we need to know the parent's tag value at runtime.
	parentTagGoName := KebabToPascal(parentResourcePath[len(parentResourcePath)-1]) + "Name"
	parentTagPulumiName := PascalToCamel(parentTagGoName)

	res := Resource{
		GoName:      goName,
		Description: helpText,
		Kind:        TagNodeResource,
		FileName:    fileName,
		TagFields: []TagField{
			{
				GoName:      parentTagGoName,
				PulumiName:  parentTagPulumiName,
				Description: "Parent tag node key",
				PathPrefix:  append([]string{}, parentResourcePath...),
			},
			{
				GoName:      "Name",
				PulumiName:  "name",
				Description: helpText,
				PathPrefix:  []string{tn.Name},
			},
		},
	}

	var subResources []Resource
	if tn.Children != nil {
		res.Fields, subResources = collectFields(tn.Children, resourcePath, resourceIsContainer, nil)
	}

	result := []Resource{res}
	result = append(result, subResources...)
	return result
}

// buildField creates a Field from a leafNode with the given prefix path.
func buildField(ln *xmlparse.LeafNode, prefix []string) Field {
	goName := BuildGoFieldName(prefix, ln.Name)
	pulumiName := PascalToCamel(goName)
	goName, pulumiName = FixReservedFieldName(goName, pulumiName)

	ft := StringField
	helpText := ""
	if ln.Properties != nil {
		ft = InferFieldType(ln.Properties)
		helpText = ln.Properties.Help
	}

	vyosPath := make([]string, 0, len(prefix)+1)
	vyosPath = append(vyosPath, prefix...)
	vyosPath = append(vyosPath, ln.Name)

	return Field{
		GoName:      goName,
		PulumiName:  pulumiName,
		VyosName:    ln.Name,
		VyosPath:    vyosPath,
		GoType:      GoTypeFor(ft),
		FieldType:   ft,
		Description: helpText,
	}
}
