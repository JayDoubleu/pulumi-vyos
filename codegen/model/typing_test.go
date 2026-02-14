package model

import (
	"testing"

	"github.com/jaydoubleu/pulumi-vyos/codegen/xmlparse"
)

func TestInferFieldType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		props *xmlparse.Properties
		want  FieldType
	}{
		{
			name:  "nil properties",
			props: nil,
			want:  StringField,
		},
		{
			name: "valueless boolean",
			props: &xmlparse.Properties{
				Help:      "Disable interface",
				Valueless: &struct{}{},
			},
			want: BoolField,
		},
		{
			name: "u32 integer",
			props: &xmlparse.Properties{
				Help: "MTU",
				ValueHelps: []xmlparse.ValueHelp{
					{Format: "u32:68-16000", Description: "MTU value"},
				},
			},
			want: IntField,
		},
		{
			name: "all u32 values",
			props: &xmlparse.Properties{
				Help: "Ring buffer",
				ValueHelps: []xmlparse.ValueHelp{
					{Format: "u32:80-16384", Description: "ring buffer size"},
				},
			},
			want: IntField,
		},
		{
			name: "mixed formats not int",
			props: &xmlparse.Properties{
				Help: "Port",
				ValueHelps: []xmlparse.ValueHelp{
					{Format: "txt", Description: "named port"},
					{Format: "u32:1-65535", Description: "numbered port"},
				},
			},
			want: StringField,
		},
		{
			name: "multi value",
			props: &xmlparse.Properties{
				Help:  "Addresses",
				Multi: &struct{}{},
			},
			want: MultiField,
		},
		{
			name: "plain string",
			props: &xmlparse.Properties{
				Help: "Description",
			},
			want: StringField,
		},
		{
			name: "valueless takes priority over multi",
			props: &xmlparse.Properties{
				Help:      "Some flag",
				Valueless: &struct{}{},
				Multi:     &struct{}{},
			},
			want: BoolField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := InferFieldType(tt.props)
			if got != tt.want {
				t.Errorf("InferFieldType() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGoTypeFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ft   FieldType
		want string
	}{
		{StringField, "*string"},
		{IntField, "*int"},
		{BoolField, "*bool"},
		{MultiField, "[]string"},
	}

	for _, tt := range tests {
		got := GoTypeFor(tt.ft)
		if got != tt.want {
			t.Errorf("GoTypeFor(%d) = %q, want %q", tt.ft, got, tt.want)
		}
	}
}
