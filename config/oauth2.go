package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &OAuth2{}

// OAuth2 represents the <OAuth2> object.
type OAuth2 struct {
	BackendName   string   `hcl:"backend,optional"`
	GrantType     string   `hcl:"grant_type"`
	Remain        hcl.Body `hcl:",remain"`
	TokenEndpoint string   `hcl:"token_endpoint"`
}

// HCLBody implements the <Inline> interface.
func (oa OAuth2) HCLBody() hcl.Body {
	return oa.Remain
}

// Reference implements the <Inline> interface.
func (oa OAuth2) Reference() string {
	return "OAuth2"
}

// Schema implements the <Inline> interface.
func (oa OAuth2) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	type Inline struct {
		Backend      *Backend `hcl:"backend,block"`
		ClientID     string   `hcl:"client_id"`
		ClientSecret string   `hcl:"client_secret"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, oa.HCLBody())
}
