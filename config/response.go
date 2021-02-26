package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Response{}

// Response represents the <Response> object.
type Response struct {
	Body   string   `hcl:"body,optional"`
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (r Response) HCLBody() hcl.Body {
	return r.Remain
}

// Reference implements the <Inline> interface.
func (r Response) Reference() string {
	return "resp"
}

// Schema implements the <Inline> interface.
func (r Response) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	type Inline struct {
		Body    string            `hcl:"body,optional"`
		Headers map[string]string `hcl:"headers,optional"`
		Status  int               `hcl:"status,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
