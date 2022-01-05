package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &Endpoint{}

// Endpoint represents the <Endpoint> object.
type Endpoint struct {
	ErrorHandlerSetter
	AccessControl        []string  `hcl:"access_control,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Pattern              string    `hcl:"pattern,label"`
	Remain               hcl.Body  `hcl:",remain"`
	RequestBodyLimit     string    `hcl:"request_body_limit,optional"`
	Response             *Response `hcl:"response,block"`
	Scope                cty.Value `hcl:"beta_scope,optional"`

	// internally configured due to multi-label options
	Proxies   Proxies
	Requests  Requests
	Sequences Sequences
}

// Endpoints represents a list of <Endpoint> objects.
type Endpoints []*Endpoint

// HCLBody implements the <Inline> interface.
func (e Endpoint) HCLBody() hcl.Body {
	return e.Remain
}

// Inline implements the <Inline> interface.
func (e Endpoint) Inline() interface{} {
	type Inline struct {
		meta.Attributes
		Proxies        Proxies                   `hcl:"proxy,block"`
		Requests       Requests                  `hcl:"request,block"`
		ResponseStatus *uint8                    `hcl:"set_response_status,optional"`
		LogFields      map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (e Endpoint) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(e.Inline())

	return meta.SchemaWithAttributes(schema)
}
