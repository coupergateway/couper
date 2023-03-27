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
	_
	JSONParseRequest
	_
	_
	_
	JSONParseResponse
)

func (i BufferOption) GoString() string {
	var result []string
	for _, o := range []BufferOption{BufferRequest, BufferResponse, JSONParseRequest, JSONParseResponse} {
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

	var allExprs []hclsyntax.Expression
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
				if rootName == ClientRequest || rootName == BackendRequests || rootName == BackendRequest {
					result |= BufferRequest
				}
				if rootName == BackendResponses || rootName == BackendResponse {
					result |= BufferResponse
				}
				continue
			}

			if rootName != ClientRequest && rootName != BackendRequests && rootName != BackendRequest && rootName != BackendResponses && rootName != BackendResponse {
				continue
			}

			for _, t := range traversal[1:] {
				nameField := reflect.ValueOf(t).FieldByName("Name")
				name := nameField.String()
				switch name {
				case Body:
					switch rootName {
					case ClientRequest:
						fallthrough
					case BackendRequest:
						fallthrough
					case BackendRequests:
						result |= BufferRequest

					case BackendResponse:
						fallthrough
					case BackendResponses:
						result |= BufferResponse
					}
				case CTX: // e.g. jwt token (value) could be read from any (body) source
					if rootName == ClientRequest || rootName == BackendRequests || rootName == BackendRequest {
						result |= BufferRequest
					}
				case FormBody:
					if rootName == ClientRequest || rootName == BackendRequests || rootName == BackendRequest {
						result |= BufferRequest
					}
				case JSONBody:
					switch rootName {
					case ClientRequest:
						fallthrough
					case BackendRequest:
						fallthrough
					case BackendRequests:
						result |= BufferRequest
						result |= JSONParseRequest

					case BackendResponse:
						fallthrough
					case BackendResponses:
						result |= BufferResponse
						result |= JSONParseResponse
					}
				default:
					// e.g. backend_responses.default
					if len(traversal) == 2 {
						if rootName == BackendResponse || rootName == BackendResponses {
							result |= BufferResponse
						}
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
