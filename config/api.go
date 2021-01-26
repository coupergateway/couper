package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &API{}

type API struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	CORS                 *CORS     `hcl:"cors,block"`
	Backend              string    `hcl:"backend,optional"`
	BasePath             string    `hcl:"base_path,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Remain               hcl.Body  `hcl:",remain" json:"-"`
}

func (a API) Body() hcl.Body {
	return a.Remain
}

func (a API) Reference() string {
	return a.Backend
}

func (a API) Schema(inline bool) *hcl.BodySchema {
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
