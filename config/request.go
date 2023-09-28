package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ BackendReference = &Request{}
	_ Body             = &Request{}
	_ Inline           = &Request{}
)

// Request represents the <Request> object.
type Request struct {
	BackendName string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the request. Mutually exclusive with {backend} block."`
	Name        string   `hcl:"name,label,optional"`
	Remain      hcl.Body `hcl:",remain"`

	// Internally used
	Backend *hclsyntax.Body
}

// Requests represents a list of <Requests> objects.
type Requests []*Request

// Reference implements the <BackendReference> interface.
func (r Request) Reference() string {
	return r.BackendName
}

// HCLBody implements the <Body> interface.
func (r Request) HCLBody() *hclsyntax.Body {
	return r.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (r Request) Inline() interface{} {
	type Inline struct {
		Backend        *Backend             `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for the request (zero or one). Mutually exclusive with {backend} attribute."`
		Body           string               `hcl:"body,optional" docs:"Plain text request body, implicitly sets {Content-Type: text/plain} header field."`
		ExpectedStatus []int                `hcl:"expected_status,optional" docs:"If defined, the response status code will be verified against this list of codes. If the status code is not included in this list an [{unexpected_status} error](../error-handling#endpoint-error-types) will be thrown which can be handled with an [{error_handler}](../error-handling#endpoint-related-error_handler)."`
		FormBody       string               `hcl:"form_body,optional" docs:"Form request body, implicitly sets {Content-Type: application/x-www-form-urlencoded} header field."`
		Headers        map[string]string    `hcl:"headers,optional" docs:"Same as {set_request_headers} in [Modifiers - Request Header](../modifiers#request-header)."`
		JSONBody       string               `hcl:"json_body,optional" docs:"JSON request body, implicitly sets {Content-Type: application/json} header field."`
		Method         string               `hcl:"method,optional" docs:"The request method." default:"GET"`
		QueryParams    map[string]cty.Value `hcl:"query_params,optional" docs:"Key/value pairs to set query parameters for this request."`
		URL            string               `hcl:"url,optional" docs:"URL of the resource to request. May be relative to an origin specified in a referenced or nested {backend} block."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (r Request) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(r.Inline())

	return schema
}
