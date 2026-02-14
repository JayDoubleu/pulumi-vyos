// Package xmlparse provides XML unmarshaling for VyOS interface definition files.
package xmlparse

import "encoding/xml"

// InterfaceDefinition is the root element of a VyOS XML interface definition file.
type InterfaceDefinition struct {
	XMLName   xml.Name   `xml:"interfaceDefinition"`
	Nodes     []Node     `xml:"node"`
	TagNodes  []TagNode  `xml:"tagNode"`
	LeafNodes []LeafNode `xml:"leafNode"`
}

// Node is an intermediate container in the VyOS config tree.
type Node struct {
	Name       string      `xml:"name,attr"`
	Owner      string      `xml:"owner,attr"`
	Properties *Properties `xml:"properties"`
	Children   *Children   `xml:"children"`
}

// TagNode is a named/dynamic container (e.g., ethernet eth0).
type TagNode struct {
	Name       string      `xml:"name,attr"`
	Owner      string      `xml:"owner,attr"`
	Properties *Properties `xml:"properties"`
	Children   *Children   `xml:"children"`
}

// LeafNode is a terminal value node.
type LeafNode struct {
	Name         string      `xml:"name,attr"`
	Owner        string      `xml:"owner,attr"`
	Properties   *Properties `xml:"properties"`
	DefaultValue *string     `xml:"defaultValue"`
}

// Children contains the child nodes of a Node or TagNode.
type Children struct {
	Nodes     []Node     `xml:"node"`
	TagNodes  []TagNode  `xml:"tagNode"`
	LeafNodes []LeafNode `xml:"leafNode"`
}

// Properties holds metadata for a node.
type Properties struct {
	Help                   string      `xml:"help"`
	Priority               *int        `xml:"priority"`
	Valueless              *struct{}   `xml:"valueless"`
	Multi                  *struct{}   `xml:"multi"`
	ValueHelps             []ValueHelp `xml:"valueHelp"`
	Constraint             *Constraint `xml:"constraint"`
	ConstraintErrorMessage string      `xml:"constraintErrorMessage"`
}

// Constraint defines validation rules for a VyOS config value.
type Constraint struct {
	Regex      []string    `xml:"regex"`
	Validators []Validator `xml:"validator"`
}

// Validator references a named VyOS validation function.
type Validator struct {
	Name     string `xml:"name,attr"`
	Argument string `xml:"argument,attr"`
}

// ValueHelp describes an accepted value format.
type ValueHelp struct {
	Format      string `xml:"format"`
	Description string `xml:"description"`
}
