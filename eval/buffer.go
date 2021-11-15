//go:generate stringer -type=BufferOption -output=./buffer_string.go

package eval

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
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

// MustBuffer determines if any of the hcl.bodies makes use of 'form_body' or 'json_body'.
func MustBuffer(body hcl.Body) BufferOption {
	result := BufferNone

	if body == nil {
		return result
	}

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
			if rootName != ClientRequest && rootName != BackendResponses {
				continue
			}

			nameField := reflect.ValueOf(traversal[1]).FieldByName("Name")
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
	return result
}
