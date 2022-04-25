package seetie

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func Test_stringListToValue(t *testing.T) {
	tests := []struct {
		slice []string
	}{
		{[]string{"a", "b"}},
		{[]string{}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.slice), func(subT *testing.T) {
			val := stringListToValue(tt.slice)
			valType := val.Type()
			if !valType.IsListType() {
				t.Error("Expected value type to be list")
			}
			if *valType.ListElementType() != cty.String {
				t.Error("Expected list to contain string values")
			}
			sl := val.AsValueSlice()
			if len(sl) != len(tt.slice) {
				t.Errorf("Wrong number of items; want: %d, got: %d", len(tt.slice), len(sl))
			}
			for i, v := range tt.slice {
				if sl[i].AsString() != v {
					t.Errorf("Wrong item at position %d; want %q, got %q", i, v, sl[i])
				}
			}
		})
	}
}
