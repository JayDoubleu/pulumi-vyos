// Command generate reads VyOS XML interface definitions and produces Go
// resource files for the Pulumi provider.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jaydoubleu/pulumi-vyos/codegen/generate"
	"github.com/jaydoubleu/pulumi-vyos/codegen/model"
	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

func main() {
	xmlDir := flag.String("xml-dir", "", "Path to VyOS interface-definitions directory")
	outputDir := flag.String("output-dir", "", "Path to write generated Go files")
	flag.Parse()

	if *xmlDir == "" || *outputDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	files, err := filepath.Glob(filepath.Join(*xmlDir, "*.xml.in"))
	if err != nil {
		log.Fatalf("glob XML files: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no .xml.in files found in %s", *xmlDir)
	}
	sort.Strings(files)

	var allResources []model.Resource
	var parseErrors int

	for _, f := range files {
		def, err := xmlparse.ParseFile(f, *xmlDir)
		if err != nil {
			log.Printf("WARN: skip %s: %v", filepath.Base(f), err)
			parseErrors++
			continue
		}
		resources := model.Build(def)
		allResources = append(allResources, resources...)
	}

	// Deduplicate resources: merge true duplicates (same API path) and
	// disambiguate naming collisions (different API paths, same GoName).
	beforeDedup := len(allResources)
	allResources = model.Deduplicate(allResources)

	// Sort resources by name for stable output.
	sort.Slice(allResources, func(i, j int) bool {
		return allResources[i].GoName < allResources[j].GoName
	})

	log.Printf("Parsed %d XML files (%d skipped), %d resources before dedup, %d after",
		len(files)-parseErrors, parseErrors, beforeDedup, len(allResources))

	if err := generate.Generate(allResources, *outputDir); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Printf("Generated %d resource files in %s\n", len(allResources), *outputDir)
}
