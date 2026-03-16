package lib

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestFlattenFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    cty.Value
		expected []string
	}{
		{
			"flat list",
			cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			[]string{"a", "b"},
		},
		{
			"nested list",
			cty.TupleVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
				cty.TupleVal([]cty.Value{cty.StringVal("c")}),
			}),
			[]string{"a", "b", "c"},
		},
		{
			"deeply nested",
			cty.TupleVal([]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.TupleVal([]cty.Value{cty.StringVal("deep")}),
				}),
			}),
			[]string{"deep"},
		},
		{
			"empty list",
			cty.EmptyTupleVal,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FlattenFunc.Call([]cty.Value{tt.input})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expected == nil {
				if result.LengthInt() != 0 {
					t.Fatalf("expected empty tuple, got length %d", result.LengthInt())
				}
				return
			}

			if result.LengthInt() != len(tt.expected) {
				t.Fatalf("expected length %d, got %d", len(tt.expected), result.LengthInt())
			}

			i := 0
			for it := result.ElementIterator(); it.Next(); {
				_, v := it.Element()
				if v.AsString() != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, v.AsString(), tt.expected[i])
				}
				i++
			}
		})
	}
}

func TestFlattenFunc_Null(t *testing.T) {
	result, err := FlattenFunc.Call([]cty.Value{cty.NullVal(cty.List(cty.String))})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LengthInt() != 0 {
		t.Errorf("expected empty tuple for null input, got length %d", result.LengthInt())
	}
}
