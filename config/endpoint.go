package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type Endpoint struct {
	AccessControl        []string `hcl:"access_control,optional"`
	Backend              string   `hcl:"backend,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	InlineDefinition     hcl.Body `hcl:",remain" json:"-"`
	Pattern              string   `hcl:"path,label"`
}

func (e Endpoint) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	type Inline struct {
		Path    string   `hcl:"path,optional"`
		Backend *Backend `hcl:"backend,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})
	return schema
}
