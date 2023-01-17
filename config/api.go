package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ Body   = &API{}
	_ Inline = &API{}
)

// API represents the <API> object.
type API struct {
	ErrorHandlerSetter
	AccessControl        []string  `hcl:"access_control,optional" docs:"Sets predefined [access control](../access-control) for this block."`
	AllowedMethods       []string  `hcl:"allowed_methods,optional" docs:"Sets allowed methods as _default_ for all contained endpoints. Requests with a method that is not allowed result in an error response with a {405 Method Not Allowed} status." default:"*"`
	BasePath             string    `hcl:"base_path,optional" docs:"Configures the path prefix for all requests."`
	CORS                 *CORS     `hcl:"cors,block" docs:"Configures [CORS](/configuration/block/cors) settings (zero or one)."`
	DisableAccessControl []string  `hcl:"disable_access_control,optional" docs:"Disables access controls by name."`
	Endpoints            Endpoints `hcl:"endpoint,block" docs:"Configures an [endpoint](/configuration/block/endpoint) (zero or more)."`
	ErrorFile            string    `hcl:"error_file,optional" docs:"Location of the error file template."`
	Name                 string    `hcl:"name,label,optional"`
	Remain               hcl.Body  `hcl:",remain"`

	// internally used
	CatchAllEndpoint   *Endpoint
	RequiredPermission hcl.Expression
}

// APIs represents a list of <API> objects.
type APIs []*API

// HCLBody implements the <Body> interface.
func (a API) HCLBody() *hclsyntax.Body {
	return a.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (a API) Inline() interface{} {
	type Inline struct {
		meta.ResponseHeadersAttributes
		meta.LogFieldsAttribute
		RequiredPermission hcl.Expression `hcl:"required_permission,optional" docs:"Permission required to use this API (see [error type](/configuration/error-handling#error-types) {beta_insufficient_permissions})." type:"string or object (string)"`
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
	return meta.MergeSchemas(schema, meta.ResponseHeadersAttributesSchema, meta.LogFieldsAttributeSchema)
}
