package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var (
	_ Body   = &Response{}
	_ Inline = &Response{}

	ResponseInlineSchema = Response{}.Schema(true)
)

// Response represents the <Response> object.
type Response struct {
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface.
func (r Response) HCLBody() *hclsyntax.Body {
	return r.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (r Response) Inline() interface{} {
	type Inline struct {
		Body     string            `hcl:"body,optional" docs:"Response body which creates implicit default {Content-Type: text/plain} header field."`
		JSONBody string            `hcl:"json_body,optional" docs:"JSON response body which creates implicit default {Content-Type: application/json} header field." type:"null, bool, number, string, object, tuple"`
		Headers  map[string]string `hcl:"headers,optional" docs:"Same as {set_response_headers} in [Modifiers - Response Header](../modifiers#response-header)."`
		Status   int               `hcl:"status,optional" docs:"The HTTP status code to return." default:"200"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (r Response) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(r.Inline())

	return schema
}
