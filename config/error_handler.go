package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &ErrorHandler{}

// ErrorHandler represents a subset of Endpoint.
type ErrorHandler struct {
	Kind      string
	ErrorFile string    `hcl:"error_file,optional"`
	Remain    hcl.Body  `hcl:",remain"`
	Response  *Response `hcl:"response,block"`

	// internally configured due to multi-label options
	Proxies  Proxies
	Requests Requests
}

// ErrorHandlerGetter defines the <ErrorHandlerGetter> interface.
type ErrorHandlerGetter interface {
	DefaultErrorHandler() *ErrorHandler
}

// HCLBody implements the <Inline> interface.
func (e ErrorHandler) HCLBody() hcl.Body {
	return e.Remain
}

// Inline implements the <Inline> interface.
func (e ErrorHandler) Inline() interface{} {
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
func (e ErrorHandler) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(e.Inline())

	return meta.SchemaWithAttributes(schema)
}
