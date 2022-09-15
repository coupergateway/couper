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
	BackendName string   `hcl:"backend,optional" docs:"{backend} block reference, defined in [{definitions}](definitions). Required, if no [{backend} block](backend) or {url} is defined within."`
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
func (r Request) HCLBody() hcl.Body {
	return r.Remain
}

// Inline implements the <Inline> interface.
func (r Request) Inline() interface{} {
	type Inline struct {
		Backend        *Backend             `hcl:"backend,block"`
		Body           string               `hcl:"body,optional" docs:"plain text request body, implicitly sets {Content-Type: text/plain} header field."`
		ExpectedStatus []int                `hcl:"expected_status,optional" docs:"If defined, the response status code will be verified against this list of codes. If the status code is not included in this list an [{unexpected_status} error](../error-handling#endpoint-error-types) will be thrown which can be handled with an [{error_handler}](../error-handling#endpoint-related-error_handler)."`
		FormBody       string               `hcl:"form_body,optional" docs:"form request body, implicitly sets {Content-Type: application/x-www-form-urlencoded} header field."`
		Headers        map[string]string    `hcl:"headers,optional" docs:"request headers"`
		JSONBody       string               `hcl:"json_body,optional" docs:"JSON request body, implicitly sets {Content-Type: application/json} header field."`
		Method         string               `hcl:"method,optional" docs:"the request method" default:"GET"`
		QueryParams    map[string]cty.Value `hcl:"query_params,optional" docs:"Key/value pairs to set query parameters for this request"`
		URL            string               `hcl:"url,optional" docs:"If defined, the host part of the URL must be the same as the {origin} attribute of the used backend."`
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

	// A backend reference is defined, backend block is not allowed.
	if r.BackendName != "" {
		schema.Blocks = nil
	}

	return schema
}
