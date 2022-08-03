package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/config/sequence"
)

var _ Inline = &Endpoint{}

// Endpoint represents the <Endpoint> object.
type Endpoint struct {
	ErrorHandlerSetter
	AccessControl        []string  `hcl:"access_control,optional" docs:"Sets predefined access control for this block context."`
	AllowedMethods       []string  `hcl:"allowed_methods,optional" docs:"Sets allowed methods overriding a default set in the containing {api} block. Requests with a method that is not allowed result in an error response with a {405 Method Not Allowed} status." default:"*"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional" docs:"Disables access controls by name."`
	ErrorFile            string    `hcl:"error_file,optional" docs:"Location of the error file template."`
	Pattern              string    `hcl:"pattern,label"`
	Proxies              Proxies   `hcl:"proxy,block"`
	Remain               hcl.Body  `hcl:",remain"`
	RequestBodyLimit     string    `hcl:"request_body_limit,optional" docs:"Configures the maximum buffer size while accessing {request.form_body} or {request.json_body} content. Valid units are: {KiB}, {MiB}, {GiB}" default:"64MiB"`
	Requests             Requests  `hcl:"request,block"`
	Response             *Response `hcl:"response,block"`

	// internally configured due to multi-label options
	RequiredPermission hcl.Expression
	Sequences          sequence.List
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
		ResponseStatus     *uint8                    `hcl:"set_response_status,optional" docs:"Modifies the response status code."`
		LogFields          map[string]hcl.Expression `hcl:"custom_log_fields,optional" docs:"Defines log fields for custom logging"`
		RequiredPermission hcl.Expression            `hcl:"beta_required_permission,optional" docs:"expression evaluating to string or object (string)"`
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
