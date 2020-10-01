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
				if rootName != "req" && rootName != "beresp" {
					continue
				}
				for _, step := range traversal[1:] {
					nameField := reflect.ValueOf(step).FieldByName("Name")
					name := nameField.String()
					switch name {
					case "json_body":
						switch rootName {
						case "req":
							result |= BufferRequest
						case "beresp":
							result |= BufferResponse
						}
					case "post":
						if rootName == "req" {
							result |= BufferRequest
						}
					}
				}
			}
		}
	}
	return result
}
