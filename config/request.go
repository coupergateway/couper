package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ BackendReference = &Request{}
	_ Inline           = &Request{}
)

// Request represents the <Request> object.
type Request struct {
	BackendName string   `hcl:"backend,optional"`
	Name        string   `hcl:"name,label"`
	Remain      hcl.Body `hcl:",remain"`
	// Internally used
	Backend hcl.Body
}

// Requests represents a list of <Requests> objects.
type Requests []*Request

// HCLBody implements the <Inline> interface.
func (r Request) HCLBody() hcl.Body {
	return r.Remain
}

// Schema implements the <Inline> interface.
func (r Request) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	type Inline struct {
		Backend     *Backend             `hcl:"backend,block"`
		Body        string               `hcl:"body,optional"`
		FormBody    string               `hcl:"form_body,optional"`
		JsonBody    string               `hcl:"json_body,optional"`
		Headers     map[string]string    `hcl:"headers,optional"`
		Method      string               `hcl:"method,optional"`
		QueryParams map[string]cty.Value `hcl:"query_params,optional"`
		URL         string               `hcl:"url,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if r.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, r.HCLBody())
}
