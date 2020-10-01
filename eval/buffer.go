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
				rootName := traversal.RootName()
				if rootName != ClientRequest && rootName != BackendResponse {
					continue
				}
				for _, step := range traversal[1:] {
					nameField := reflect.ValueOf(step).FieldByName("Name")
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
	}
	return result
}
