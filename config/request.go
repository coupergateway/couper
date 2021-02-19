package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &Request{}

// Request represents the <Request> object.
type Request struct {
	Backend string   `hcl:"backend,optional"`
	Body    string   `hcl:"body,optional"`
	Method  string   `hcl:"method,optional"`
	Name    string   `hcl:"name,label"`
	Remain  hcl.Body `hcl:",remain"`
	URL     string   `hcl:"url,optional"`
}

// Requests represents a list of <Requests> objects.
type Requests []*Request

// HCLBody implements the <Inline> interface.
func (r Request) HCLBody() hcl.Body {
	return r.Remain
}

// Reference implements the <Inline> interface.
func (r Request) Reference() string {
	return r.Backend
}

// Schema implements the <Inline> interface.
func (r Request) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	type Inline struct {
		Backend     *Backend             `hcl:"backend,block"`
		Headers     map[string]string    `hcl:"headers,optional"`
		QueryParams map[string]cty.Value `hcl:"query_params,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if r.Backend != "" {
		schema.Blocks = nil
	}

	// TODO: <URL> vs. <Origin> + <Path> im <Backend>?

	return newBackendSchema(schema, r.HCLBody())
}
