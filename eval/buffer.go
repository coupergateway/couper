//go:generate stringer -type=BufferOption -output=./buffer_string.go

package eval

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"

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

// MustBuffer determines if any of the hcl.bodies makes use of 'body', 'form_body' or 'json_body' or
// of known attributes and variables which require a parsed client-request or backend-response body.
func MustBuffer(bodies ...hcl.Body) BufferOption {
	result := BufferNone

	if len(bodies) == 0 {
		return result
	}

	var allExprs []hcl.Expression
	allAttributes := body.CollectAttributes(bodies...)
	allBlockTypes := body.CollectBlockTypes(bodies...)

	for _, blockType := range allBlockTypes {
		if opt := bufferWithBlock(blockType); opt != BufferNone {
			result |= opt
		}
	}

	// TODO: follow func call and their referenced remains
	for _, attr := range allAttributes {
		allExprs = append(allExprs, attr.Expr)

		if opt := bufferWithAttribute(attr.Name); opt != BufferNone {
			result |= opt
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
				case Body:
					switch rootName {
					case ClientRequest:
						result |= BufferRequest
					case BackendResponses:
						result |= BufferResponse
					}
				case CTX: // e.g. jwt token (value) could be read from any (body) source
					if rootName == ClientRequest {
						result |= BufferRequest
					}
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
				default:
					// e.g. backend_responses.default
					if rootName == BackendResponses && len(traversal) == 2 {
						result |= BufferResponse
					}
				}
			}
		}
	}
	return result
}

func bufferWithAttribute(attrName string) BufferOption {
	switch attrName {
	case attrAddFormParams, attrSetFormParams, attrDelFormParams:
		return BufferRequest
	}
	return BufferNone
}

func bufferWithBlock(name string) BufferOption {
	switch name {
	case "openapi":
		return BufferResponse
	}
	return BufferNone
}
