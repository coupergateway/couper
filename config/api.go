package config

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type Api struct {
	AccessControl []string    `hcl:"access_control,optional"`
	BasePath      string      `hcl:"base_path,optional"`
	Backend       []*Backend  `hcl:"backend,block"`
	Endpoint      []*Endpoint `hcl:"endpoint,block"`
	PathHandler   PathHandler
}

type PathHandler map[*Endpoint]http.Handler

func (api *Api) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(api)
	if !inline {
		return schema
	}
	// backend, remove 2nd label for inline usage
	for i, block := range schema.Blocks {
		if block.Type == "backend" && len(block.LabelNames) > 1 {
			schema.Blocks[i].LabelNames = block.LabelNames[:1]
		}
	}
	return schema
}
