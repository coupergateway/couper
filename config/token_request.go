package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ BackendReference = &TokenRequest{}
	_ Inline           = &TokenRequest{}
)

var TokenRequestBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "token_request",
		},
	},
}

type TokenRequest struct {
	BackendName string   `hcl:"backend,optional"`
	URL         string   `hcl:"url"`
	Remain      hcl.Body `hcl:",remain"`

	// Internally used
	Backend hcl.Body
}

// Reference implements the <BackendReference> interface.
func (t TokenRequest) Reference() string {
	return t.BackendName
}

// HCLBody implements the <Inline> interface.
func (t TokenRequest) HCLBody() hcl.Body {
	return t.Remain
}

// Inline implements the <Inline> interface.
func (t TokenRequest) Inline() interface{} {
	type Inline struct {
		Backend        *Backend             `hcl:"backend,block"`
		Body           string               `hcl:"body,optional"`
		ExpectedStatus []int                `hcl:"expected_status,optional"`
		FormBody       string               `hcl:"form_body,optional"`
		Headers        map[string]string    `hcl:"headers,optional"`
		JsonBody       string               `hcl:"json_body,optional"`
		Method         string               `hcl:"method,optional"`
		QueryParams    map[string]cty.Value `hcl:"query_params,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (t TokenRequest) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(t)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(t.Inline())

	// A backend reference is defined, backend block is not allowed.
	if t.BackendName != "" {
		schema.Blocks = nil
	}

	return schema
}
