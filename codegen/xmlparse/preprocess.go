package xmlparse

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var includeRe = regexp.MustCompile(`^\s*#include\s+<(.+)>\s*$`)

// Preprocess reads an XML file, recursively resolves all #include directives,
// and returns the fully expanded content. Include paths are resolved relative
// to rootDir (typically the interface-definitions/ directory).
func Preprocess(filePath, rootDir string) ([]byte, error) {
	return preprocess(filePath, rootDir, make(map[string]bool))
}

// preprocess resolves includes with a stack-based cycle detector. The stack map
// tracks files currently being processed; entries are removed on return so that
// the same fragment can be included from different branches without false positives.
func preprocess(filePath, rootDir string, stack map[string]bool) ([]byte, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("resolve path %s: %w", filePath, err)
	}
	if stack[absPath] {
		return nil, fmt.Errorf("circular include detected: %s", filePath)
	}
	stack[absPath] = true
	defer delete(stack, absPath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if m := includeRe.FindStringSubmatch(line); m != nil {
			includePath := filepath.Join(rootDir, m[1])
			included, err := preprocess(includePath, rootDir, stack)
			if err != nil {
				return nil, fmt.Errorf("include %s from %s: %w", m[1], filePath, err)
			}
			result.Write(included)
		} else {
			result.WriteString(line)
			result.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", filePath, err)
	}

	return result.Bytes(), nil
}
