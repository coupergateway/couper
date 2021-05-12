package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ Inline = &Response{}

	ResponseInlineSchema = Response{}.Schema(true)
)

// Response represents the <Response> object.
type Response struct {
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (r Response) HCLBody() hcl.Body {
	return r.Remain
}

// Schema implements the <Inline> interface.
func (r Response) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	type Inline struct {
		Body     string            `hcl:"body,optional"`
		JsonBody string            `hcl:"json_body,optional"`
		Headers  map[string]string `hcl:"headers,optional"`
		Status   int               `hcl:"status,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
