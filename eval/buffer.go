//go:generate stringer -type=BufferOption -output=./buffer_string.go

package eval

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/body"
)

type BufferOption uint8

const (
	BufferNone BufferOption = iota
	BufferRequest
	BufferResponse
)

func (i BufferOption) GoString() string {
	var result []string
	for _, o := range []BufferOption{BufferRequest, BufferResponse} {
		if (i & o) == o {
			result = append(result, o.String())
		}
	}
	if len(result) == 0 {
		return BufferNone.String()
	}
	return strings.Join(result, "|")
}

func (i BufferOption) Response() bool {
	return i&BufferResponse == BufferResponse
}

// MustBuffer determines if any of the hcl.bodies makes use of 'form_body' or 'json_body'.
func MustBuffer(bodies ...hcl.Body) BufferOption {
	result := BufferNone

	if len(bodies) == 0 {
		return result
	}

	var allExprs []hcl.Expression
	var syntaxAttrs []hclsyntax.Attributes
	// TODO: follow func call and their referenced remains
	for _, b := range bodies {
		if sb, ok := b.(*hclsyntax.Body); ok {
			syntaxAttrs = append(syntaxAttrs, sb.Attributes)
			for _, block := range sb.Blocks {
				syntaxAttrs = append(syntaxAttrs, block.Body.Attributes)
			}
			continue
		}

		if all, ok := b.(body.Attributes); ok {
			attrs := all.JustAllAttributes()
			for _, attr := range attrs {
				for _, v := range attr {
					allExprs = append(allExprs, v.Expr)
				}
			}
		}
	}

	for _, attr := range syntaxAttrs {
		for _, v := range attr {
			allExprs = append(allExprs, v.Expr)
		}
	}

	for _, expr := range allExprs {
		for _, traversal := range expr.Variables() {
			rootName := traversal.RootName()

			if len(traversal) == 1 {
				if rootName == ClientRequest {
					result |= BufferRequest
				}
				if rootName == BackendResponses {
					result |= BufferResponse
				}
				continue
			}

			if rootName != ClientRequest && rootName != BackendResponses {
				continue
			}

			for _, t := range traversal[1:] {
				nameField := reflect.ValueOf(t).FieldByName("Name")
				name := nameField.String()
				switch name {
				case FormBody:
					if rootName == ClientRequest {
						result |= BufferRequest
					}
				case JsonBody:
					switch rootName {
					case ClientRequest:
						result |= BufferRequest
					case BackendResponses:
						result |= BufferResponse
					}
				}
			}
		}
	}
	return result
}
