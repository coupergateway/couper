package configload

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestErrorHandler_newKindsFromLabels(t *testing.T) {
	b := &hclsyntax.Block{
		Labels: []string{""},
		LabelRanges: []hcl.Range{
			{
				Start: hcl.Pos{
					Line:   123,
					Column: 321,
				},
				End: hcl.Pos{
					Line:   123,
					Column: 323,
				},
			},
		},
	}

	_, err := newKindsFromLabels(b)

	exp := `message:":123,321-323: empty error_handler label; ", synopsis:"configuration error"`
	if got := fmt.Sprintf("%#v", err); !strings.Contains(got, exp) {
		t.Errorf("Unexpected error message given: %s", got)
	}
}
