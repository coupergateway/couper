package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Server{}

// Server represents the HCL <server> block.
type Server struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	API                  *Api      `hcl:"api,block"`
	Backend              string    `hcl:"backend,optional"`
	BasePath             string    `hcl:"base_path,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Files                *Files    `hcl:"files,block"`
	Hosts                []string  `hcl:"hosts,optional"`
	Name                 string    `hcl:"name,label"`
	Remain               hcl.Body  `hcl:",remain"`
	Spa                  *Spa      `hcl:"spa,block"`
}

func (s Server) Body() hcl.Body {
	return s.Remain
}

func (s Server) Reference() string {
	return s.Backend
}

func (s Server) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(s)
		return schema
	}

	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}
	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// The API contains a backend reference, backend block is not allowed.
	if s.Backend != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, s.Body())
}
