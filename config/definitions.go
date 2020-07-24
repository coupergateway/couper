package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type Definitions struct {
	Backend []*Backend `hcl:"backend,block"`
	JWT     []*JWT     `hcl:"jwt,block"`
}

func (d Definitions) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(d)
	if !inline {
		return schema
	}
	// backend, remove label for inline usage
	for i, block := range schema.Blocks {
		if block.Type == "backend" && len(block.LabelNames) > 0 {
			schema.Blocks[i].LabelNames = nil
		}
	}
	return schema
}
