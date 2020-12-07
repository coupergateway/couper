//go:generate stringer -type=BufferOption -output=./buffer_string.go

package eval

import (
	"reflect"

	"github.com/hashicorp/hcl/v2"
)

type BufferOption uint8

const (
	BufferNone BufferOption = iota
	BufferRequest
	BufferResponse
)

func (b BufferOption) Has(other BufferOption) bool {
	return (b & 1 << other) > 0
}

// MustBuffer determines if any of the hcl.bodies makes use of 'post' or 'json_body'.
func MustBuffer(ctxBodies []hcl.Body) BufferOption {
	result := BufferNone
	for _, body := range ctxBodies {
		attrs, err := body.JustAttributes()
		if err != nil {
			return result
		}
		for _, attr := range attrs {
			for _, traversal := range attr.Expr.Variables() {
				if len(traversal) < 2 {
					continue
				}

				rootName := traversal.RootName()
				if rootName != ClientRequest && rootName != BackendResponse {
					continue
				}

				nameField := reflect.ValueOf(traversal[1]).FieldByName("Name")
				name := nameField.String()
				switch name {
				case JsonBody:
					switch rootName {
					case ClientRequest:
						result |= BufferRequest
					case BackendResponse:
						result |= BufferResponse
					}
				case Post:
					if rootName == ClientRequest {
						result |= BufferRequest
					}
				}
			}
		}
	}
	return result
}
