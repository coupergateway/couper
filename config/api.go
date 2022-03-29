package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &API{}

// API represents the <API> object.
type API struct {
	ErrorHandlerSetter
	AccessControl        []string  `hcl:"access_control,optional"`
	AllowedMethods       []string  `hcl:"allowed_methods,optional"`
	BasePath             string    `hcl:"base_path,optional"`
	CORS                 *CORS     `hcl:"cors,block"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Name                 string    `hcl:"name,label"`
	Remain               hcl.Body  `hcl:",remain"`
	Scope                cty.Value `hcl:"beta_scope,optional"`

	// internally used
	CatchAllEndpoint *Endpoint
}

// APIs represents a list of <API> objects.
type APIs []*API

// HCLBody implements the <Inline> interface.
func (a API) HCLBody() hcl.Body {
	return a.Remain
}

// Inline implements the <Inline> interface.
func (a API) Inline() interface{} {
	type Inline struct {
		AddResponseHeaders map[string]string         `hcl:"add_response_headers,optional"`
		DelResponseHeaders []string                  `hcl:"remove_response_headers,optional"`
		SetResponseHeaders map[string]string         `hcl:"set_response_headers,optional"`
		LogFields          map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (a API) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(a)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(a.Inline())

	return schema
}
