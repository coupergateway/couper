package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/config/schema"
)

var (
	_ Body              = &ErrorHandler{}
	_ schema.BodySchema = &ErrorHandler{}
)

// ErrorHandler represents a subset of Endpoint.
type ErrorHandler struct {
	Kinds     []string
	ErrorFile string    `hcl:"error_file,optional" docs:"Location of the error file template."`
	Proxies   Proxies   `hcl:"proxy,block" docs:"Configures a [proxy](/configuration/block/proxy) (zero or more)."`
	Remain    hcl.Body  `hcl:",remain"`
	Requests  Requests  `hcl:"request,block" docs:"Configures a [request](/configuration/block/request) (zero or more)."`
	Response  *Response `hcl:"response,block" docs:"Configures the [response](/configuration/block/response) (zero or one)."`
}

// ErrorHandlerGetter defines the <ErrorHandlerGetter> interface.
type ErrorHandlerGetter interface {
	DefaultErrorHandler() *ErrorHandler
}

// HCLBody implements the <Body> interface.
func (e ErrorHandler) HCLBody() *hclsyntax.Body {
	return e.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (e ErrorHandler) Inline() interface{} {
	type Inline struct {
		ResponseStatus *uint8 `hcl:"set_response_status,optional"`
	}

	return &Inline{}
}

func (e ErrorHandler) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(e)
	i, _ := gohcl.ImpliedBodySchema(e.Inline())
	return meta.MergeSchemas(s, i, meta.ModifierAttributesSchema, meta.LogFieldsAttributeSchema)
}
