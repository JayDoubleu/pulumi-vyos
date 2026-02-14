package xmlparse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreprocess_SimpleInclude(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(testdataDir(t), "preprocess")
	rootDir := dir

	data, err := Preprocess(filepath.Join(dir, "main.xml.in"), rootDir)
	if err != nil {
		t.Fatalf("Preprocess error: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, `<leafNode name="description">`) {
		t.Error("expected included leafNode in output")
	}
	if strings.Contains(result, "#include") {
		t.Error("include directive should have been resolved")
	}
}

func TestPreprocess_NestedInclude(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(testdataDir(t), "preprocess")
	rootDir := dir

	data, err := Preprocess(filepath.Join(dir, "nested.xml.in"), rootDir)
	if err != nil {
		t.Fatalf("Preprocess error: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, `<node name="nested">`) {
		t.Error("expected nested node in output")
	}
	if !strings.Contains(result, `<leafNode name="description">`) {
		t.Error("expected deeply nested leafNode in output")
	}
	if strings.Contains(result, "#include") {
		t.Error("all include directives should have been resolved")
	}
}

func TestPreprocess_MissingInclude(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `<?xml version="1.0"?>
<interfaceDefinition>
  #include <include/nonexistent.xml.i>
</interfaceDefinition>
`
	mainFile := filepath.Join(dir, "test.xml.in")
	if err := os.WriteFile(mainFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Preprocess(mainFile, dir)
	if err == nil {
		t.Fatal("expected error for missing include file")
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	// Go tests run with working directory set to the package directory.
	// The testdata is at ../testdata/ relative to codegen/xmlparse/.
	dir := filepath.Join("..", "testdata")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata directory not found: %v", err)
	}
	return dir
}
