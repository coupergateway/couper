package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Spa{}

// Spa represents the <Spa> object.
type Spa struct {
	AccessControl        []string `hcl:"access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	BootstrapFile        string   `hcl:"bootstrap_file"`
	CORS                 *CORS    `hcl:"cors,block"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	Paths                []string `hcl:"paths"`
	Remain               hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (s Spa) HCLBody() hcl.Body {
	return s.Remain
}

// Schema implements the <Inline> interface.
func (s Spa) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(s)
		return schema
	}

	type Inline struct {
		AddResponseHeaders map[string]string         `hcl:"add_response_headers,optional"`
		DelResponseHeaders []string                  `hcl:"remove_response_headers,optional"`
		SetResponseHeaders map[string]string         `hcl:"set_response_headers,optional"`
		LogFields          map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
