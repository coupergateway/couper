package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type Plugin struct {
	File string `hcl:"file"`
	Name string `hcl:"name,label"`
}

func (p Plugin) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(p)
	return s
}
