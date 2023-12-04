//go:generate stringer -type=Option -output=./option_string.go

package buffer

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/eval/attributes"
	"github.com/coupergateway/couper/eval/variables"
)

type Option uint8

const (
	None              Option = 0
	Request           Option = 1
	Response          Option = 2
	JSONParseRequest  Option = 4
	JSONParseResponse Option = 8
)

func (i Option) Request() bool {
	return i&Request == Request
}

func (i Option) JSONRequest() bool {
	return i&JSONParseRequest == JSONParseRequest
}

func (i Option) Response() bool {
	return i&Response == Response
}

func (i Option) JSONResponse() bool {
	return i&JSONParseResponse == JSONParseResponse
}

func (i Option) GoString() string {
	var result []string
	for _, o := range []Option{Request, Response, JSONParseRequest, JSONParseResponse} {
		if (i & o) == o {
			result = append(result, o.String())
		}
	}
	if len(result) == 0 {
		return None.String()
	}
	return strings.Join(result, "|")
}

// Must determine if any of the hcl.bodies makes use of 'body', 'form_body' or 'json_body' or
// of known attributes and variables which require a parsed client-request or backend-response body.
func Must(bodies ...hcl.Body) Option {
	result := None

	if len(bodies) == 0 {
		return result
	}

	var allExprs []hclsyntax.Expression
	allAttributes := body.CollectAttributes(bodies...)
	allBlockTypes := body.CollectBlockTypes(bodies...)

	for _, blockType := range allBlockTypes {
		if opt := bufferWithBlock(blockType); opt != None {
			result |= opt
		}
	}

	// TODO: follow func call and their referenced remains
	for _, attr := range allAttributes {
		allExprs = append(allExprs, attr.Expr)

		if opt := bufferWithAttribute(attr.Name); opt != None {
			result |= opt
		}
	}

	for _, expr := range allExprs {
		for _, traversal := range expr.Variables() {
			rootName := traversal.RootName()

			if len(traversal) == 1 {
				if rootName == variables.ClientRequest || rootName == variables.BackendRequest || rootName == variables.BackendRequests {
					result |= Request
				}
				if rootName == variables.BackendResponses || rootName == variables.BackendResponse {
					result |= Response
				}
				continue
			}

			if rootName != variables.ClientRequest && rootName != variables.BackendRequest && rootName != variables.BackendRequests &&
				rootName != variables.BackendResponse && rootName != variables.BackendResponses {
				continue
			}

			for _, t := range traversal[1:] {
				nameField := reflect.ValueOf(t).FieldByName("Name")
				name := nameField.String()
				switch name {
				case variables.Body:
					switch rootName {
					case variables.ClientRequest:
						fallthrough
					case variables.BackendRequest:
						fallthrough
					case variables.BackendRequests:
						result |= Request
					case variables.BackendResponse:
						fallthrough
					case variables.BackendResponses:
						result |= Response
					}
				case variables.CTX: // e.g. jwt token (value) could be read from any (body) source
					if rootName == variables.ClientRequest || rootName == variables.BackendRequests || rootName == variables.BackendRequest {
						result |= Request
					}
				case variables.FormBody:
					if rootName == variables.ClientRequest || rootName == variables.BackendRequests || rootName == variables.BackendRequest {
						result |= Request
					}
				case variables.JSONBody:
					switch rootName {
					case variables.ClientRequest:
						fallthrough
					case variables.BackendRequest:
						fallthrough
					case variables.BackendRequests:
						result |= Request
						result |= JSONParseRequest
					case variables.BackendResponse:
						fallthrough
					case variables.BackendResponses:
						result |= Response
						result |= JSONParseResponse
					}
				default:
					// e.g. backend_responses.default
					if len(traversal) == 2 {
						if rootName == variables.BackendResponse || rootName == variables.BackendResponses {
							result |= Response
						}
						if rootName == variables.BackendRequest || rootName == variables.BackendRequests {
							result |= Request
						}
					}
				}
			}
		}
	}

	return result
}

func bufferWithAttribute(attrName string) Option {
	switch attrName {
	case attributes.AddFormParams, attributes.SetFormParams, attributes.DelFormParams:
		return Request
	}
	return None
}

func bufferWithBlock(name string) Option {
	switch name {
	case "openapi":
		return Response
	}
	return None
}
