package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

// ErrorHandler represents a subset of Endpoint.
type ErrorHandler struct {
	Kinds     []string
	ErrorFile string    `hcl:"error_file,optional"`
	Remain    hcl.Body  `hcl:",remain"`
	Response  *Response `hcl:"response,block"`
	// internally configured due to multi-label options
	Proxies  Proxies
	Requests Requests
}

type ErrorHandlerGetter interface {
	DefaultErrorHandler() *ErrorHandler
}

// HCLBody implements the <Inline> interface.
func (e ErrorHandler) HCLBody() hcl.Body {
	return e.Remain
}

// Schema implements the <Inline> interface.
func (e ErrorHandler) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	type Inline struct {
		meta.Attributes
		ResponseStatus int      `hcl:"set_response_status"`
		Proxies        Proxies  `hcl:"proxy,block"`
		Requests       Requests `hcl:"request,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
