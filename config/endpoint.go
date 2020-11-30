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
		Backend *Backend `hcl:"backend,block"`
		Path    string   `hcl:"path,optional"`
	}
	schema, _ := gohcl.ImpliedBodySchema(&Inline{})
	for i, block := range schema.Blocks {
		// inline backend block MAY have no label
		if block.Type == "backend" && len(block.LabelNames) > 0 {
			schema.Blocks[i].LabelNames = nil
		}
	}

	// The endpoint contains a backend reference, backend block is not allowed.
	if e.Backend != "" {
		schema.Blocks = nil
	}

	return schema
}
