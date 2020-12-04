package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Api{}

type Api struct {
	AccessControl        []string    `hcl:"access_control,optional"`
	CORS                 *CORS       `hcl:"cors,block"`
	Backend              string      `hcl:"backend,optional"`
	BasePath             string      `hcl:"base_path,optional"`
	DisableAccessControl []string    `hcl:"disable_access_control,optional"`
	Endpoint             []*Endpoint `hcl:"endpoint,block"`
	ErrorFile            string      `hcl:"error_file,optional"`
	Remain               hcl.Body    `hcl:",remain" json:"-"`
}

func (a Api) Body() hcl.Body {
	return a.Remain
}

func (a Api) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(a)
		return schema
	}

	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}
	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// The API contains a backend reference, backend block is not allowed.
	if a.Backend != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, a.Body())
}
