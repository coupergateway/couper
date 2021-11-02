package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Server{}

// Server represents the <Server> object.
type Server struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	BasePath             string    `hcl:"base_path,optional"`
	CORS                 *CORS     `hcl:"cors,block"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Files                *Files    `hcl:"files,block"`
	Hosts                []string  `hcl:"hosts,optional"`
	Name                 string    `hcl:"name,label"`
	Remain               hcl.Body  `hcl:",remain"`
	Spa                  *Spa      `hcl:"spa,block"`
	APIs                 APIs
}

// Servers represents a list of <Server> objects.
type Servers []*Server

// HCLBody implements the <Inline> interface.
func (s Server) HCLBody() hcl.Body {
	return s.Remain
}

// Schema implements the <Inline> interface.
func (s Server) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(s)
		return schema
	}

	type Inline struct {
		AddResponseHeaders map[string]string `hcl:"add_response_headers,optional"`
		DelResponseHeaders []string          `hcl:"remove_response_headers,optional"`
		SetResponseHeaders map[string]string `hcl:"set_response_headers,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
