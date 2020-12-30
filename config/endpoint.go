package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &Endpoint{}

type Endpoint struct {
	AccessControl        []string `hcl:"access_control,optional"`
	Backend              string   `hcl:"backend,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	Pattern              string   `hcl:"path,label"`
	Remain               hcl.Body `hcl:",remain" json:"-"`
}

type Endpoints []*Endpoint

func (e Endpoint) Body() hcl.Body {
	return e.Remain
}

func (e Endpoint) Reference() string {
	return e.Backend
}

func (e Endpoint) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	type Inline struct {
		Backend        *Backend             `hcl:"backend,block"`
		Path           string               `hcl:"path,optional"`
		AddQueryParams map[string]cty.Value `hcl:"add_query_params,optional"`
		DelQueryParams []string             `hcl:"remove_query_params,optional"`
		SetQueryParams map[string]cty.Value `hcl:"set_query_params,optional"`
	}
	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// The endpoint contains a backend reference, backend block is not allowed.
	if e.Backend != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, e.Body())
}
