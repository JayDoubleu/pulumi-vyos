package xmlparse

import (
	"encoding/xml"
	"fmt"
)

// ParseFile preprocesses and unmarshals a VyOS XML interface definition file.
// rootDir is the directory containing include files (typically interface-definitions/).
func ParseFile(filePath, rootDir string) (*InterfaceDefinition, error) {
	data, err := Preprocess(filePath, rootDir)
	if err != nil {
		return nil, fmt.Errorf("preprocess %s: %w", filePath, err)
	}
	return Parse(data)
}

// Parse unmarshals already-preprocessed XML content into an InterfaceDefinition.
func Parse(data []byte) (*InterfaceDefinition, error) {
	var def InterfaceDefinition
	if err := xml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("unmarshal XML: %w", err)
	}
	return &def, nil
}
