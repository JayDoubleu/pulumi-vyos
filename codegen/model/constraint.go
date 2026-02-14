package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

// BuildConstraint extracts usable validation constraints from XML properties.
// Returns nil if no regex patterns or numeric ranges are found.
func BuildConstraint(props *xmlparse.Properties) *FieldConstraint {
	if props == nil || props.Constraint == nil {
		return nil
	}

	c := props.Constraint
	var fc FieldConstraint

	for _, pattern := range c.Regex {
		// Skip patterns using Perl-specific syntax that Go's regexp
		// engine does not support (lookahead, lookbehind, etc.).
		if _, err := regexp.Compile(pattern); err != nil {
			continue
		}
		fc.Patterns = append(fc.Patterns, pattern)
	}

	for _, v := range c.Validators {
		if v.Name == "numeric" && v.Argument != "" {
			r, err := ParseNumericRange(v.Argument)
			if err == nil {
				fc.NumericRange = r
			}
		}
	}

	if len(fc.Patterns) == 0 && fc.NumericRange == nil {
		return nil
	}

	fc.ErrorMessage = props.ConstraintErrorMessage
	return &fc
}

// ParseNumericRange parses a VyOS numeric validator argument like "--range 1-255".
func ParseNumericRange(argument string) (*Range, error) {
	arg := strings.TrimSpace(argument)
	if !strings.HasPrefix(arg, "--range ") {
		return nil, fmt.Errorf("unsupported numeric argument: %s", arg)
	}
	rangeStr := strings.TrimPrefix(arg, "--range ")
	parts := strings.SplitN(rangeStr, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format: %s", rangeStr)
	}

	min, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse min %q: %w", parts[0], err)
	}
	max, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse max %q: %w", parts[1], err)
	}

	return &Range{Min: min, Max: max}, nil
}
