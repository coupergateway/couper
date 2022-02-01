package configload

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestErrorHandler_newKindsFromLabels(t *testing.T) {
	b := &hcl.Block{
		Labels: []string{""},
		DefRange: hcl.Range{
			Start: hcl.Pos{
				Line:   123,
				Column: 321,
			},
			End: hcl.Pos{
				Line:   1234,
				Column: 4321,
			},
		},
	}

	_, err := newKindsFromLabels(b)

	exp := `message:":123,321-1234,4321: invalid error_handler label format: []; ", synopsis:"configuration error"`
	if got := fmt.Sprintf("%#v", err); !strings.Contains(got, exp) {
		t.Errorf("Unexpected error message given: %s", got)
	}
}
