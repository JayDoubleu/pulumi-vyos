package model

import (
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

func TestParseNumericRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arg     string
		wantMin int64
		wantMax int64
		wantErr bool
	}{
		{"--range 1-255", 1, 255, false},
		{"--range 80-16384", 80, 16384, false},
		{"--range 0-4294967295", 0, 4294967295, false},
		{"--something else", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tt := range tests {
		r, err := ParseNumericRange(tt.arg)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseNumericRange(%q) expected error", tt.arg)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseNumericRange(%q) unexpected error: %v", tt.arg, err)
			continue
		}
		if r.Min != tt.wantMin || r.Max != tt.wantMax {
			t.Errorf("ParseNumericRange(%q) = {%d, %d}, want {%d, %d}",
				tt.arg, r.Min, r.Max, tt.wantMin, tt.wantMax)
		}
	}
}

func TestBuildConstraint(t *testing.T) {
	t.Parallel()

	t.Run("nil properties", func(t *testing.T) {
		if got := BuildConstraint(nil); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("no constraint", func(t *testing.T) {
		props := &xmlparse.Properties{Help: "test"}
		if got := BuildConstraint(props); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("regex only", func(t *testing.T) {
		props := &xmlparse.Properties{
			Constraint: &xmlparse.Constraint{
				Regex: []string{`(auto|half|full)`},
			},
			ConstraintErrorMessage: "must be auto, half or full",
		}
		fc := BuildConstraint(props)
		if fc == nil {
			t.Fatal("expected non-nil constraint")
		}
		if len(fc.Patterns) != 1 || fc.Patterns[0] != `(auto|half|full)` {
			t.Errorf("Patterns = %v", fc.Patterns)
		}
		if fc.NumericRange != nil {
			t.Errorf("unexpected NumericRange: %+v", fc.NumericRange)
		}
		if fc.ErrorMessage != "must be auto, half or full" {
			t.Errorf("ErrorMessage = %q", fc.ErrorMessage)
		}
	})

	t.Run("numeric range", func(t *testing.T) {
		props := &xmlparse.Properties{
			Constraint: &xmlparse.Constraint{
				Validators: []xmlparse.Validator{
					{Name: "numeric", Argument: "--range 80-16384"},
				},
			},
		}
		fc := BuildConstraint(props)
		if fc == nil {
			t.Fatal("expected non-nil constraint")
		}
		if fc.NumericRange == nil {
			t.Fatal("expected non-nil NumericRange")
		}
		if fc.NumericRange.Min != 80 || fc.NumericRange.Max != 16384 {
			t.Errorf("NumericRange = {%d, %d}", fc.NumericRange.Min, fc.NumericRange.Max)
		}
	})

	t.Run("non-numeric validator ignored", func(t *testing.T) {
		props := &xmlparse.Properties{
			Constraint: &xmlparse.Constraint{
				Validators: []xmlparse.Validator{
					{Name: "ipv4-address"},
				},
			},
		}
		if got := BuildConstraint(props); got != nil {
			t.Errorf("expected nil for unsupported validator, got %+v", got)
		}
	})

	t.Run("perl-only regex skipped", func(t *testing.T) {
		props := &xmlparse.Properties{
			Constraint: &xmlparse.Constraint{
				Regex: []string{`(?!-)[-a-z]+(?<!\.)`},
			},
		}
		if got := BuildConstraint(props); got != nil {
			t.Errorf("expected nil for unsupported Perl regex, got %+v", got)
		}
	})

	t.Run("mixed regex keeps valid", func(t *testing.T) {
		props := &xmlparse.Properties{
			Constraint: &xmlparse.Constraint{
				Regex: []string{`(?!-)bad`, `[a-z]+`},
			},
		}
		fc := BuildConstraint(props)
		if fc == nil {
			t.Fatal("expected non-nil constraint")
		}
		if len(fc.Patterns) != 1 || fc.Patterns[0] != `[a-z]+` {
			t.Errorf("Patterns = %v, want [\"[a-z]+\"]", fc.Patterns)
		}
	})

	t.Run("regex and numeric combined", func(t *testing.T) {
		props := &xmlparse.Properties{
			Constraint: &xmlparse.Constraint{
				Regex: []string{`[0-9]+`},
				Validators: []xmlparse.Validator{
					{Name: "numeric", Argument: "--range 1-255"},
				},
			},
			ConstraintErrorMessage: "value out of range",
		}
		fc := BuildConstraint(props)
		if fc == nil {
			t.Fatal("expected non-nil constraint")
		}
		if len(fc.Patterns) != 1 {
			t.Errorf("expected 1 pattern, got %d", len(fc.Patterns))
		}
		if fc.NumericRange == nil || fc.NumericRange.Min != 1 || fc.NumericRange.Max != 255 {
			t.Errorf("unexpected NumericRange: %+v", fc.NumericRange)
		}
	})
}

func TestHasConstraints(t *testing.T) {
	t.Parallel()

	t.Run("no constraints", func(t *testing.T) {
		r := &Resource{
			TagFields: []TagField{{GoName: "Name"}},
			Fields:    []Field{{GoName: "Desc"}},
		}
		if r.HasConstraints() {
			t.Error("expected false")
		}
	})

	t.Run("tag field constraint", func(t *testing.T) {
		r := &Resource{
			TagFields: []TagField{{
				GoName:     "Name",
				Constraint: &FieldConstraint{Patterns: []string{`eth[0-9]+`}},
			}},
		}
		if !r.HasConstraints() {
			t.Error("expected true")
		}
	})

	t.Run("field constraint", func(t *testing.T) {
		r := &Resource{
			Fields: []Field{{
				GoName:     "MTU",
				Constraint: &FieldConstraint{NumericRange: &Range{Min: 68, Max: 16000}},
			}},
		}
		if !r.HasConstraints() {
			t.Error("expected true")
		}
	})
}
