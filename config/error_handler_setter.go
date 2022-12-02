package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type ErrorHandlerSetter struct {
	ErrorHandler []*ErrorHandler `hcl:"error_handler,block" docs:"Configures an [error handler](/configuration/block/error_handler)."`
}

func (ehs *ErrorHandlerSetter) Set(ehConf *ErrorHandler) {
	ehs.ErrorHandler = append(ehs.ErrorHandler, ehConf)
}

func WithErrorHandlerSchema(schema *hcl.BodySchema) *hcl.BodySchema {
	errorSetterSchema, _ := gohcl.ImpliedBodySchema(&ErrorHandlerSetter{})
	schema.Attributes = append(schema.Attributes, errorSetterSchema.Attributes...)
	schema.Blocks = append(schema.Blocks, errorSetterSchema.Blocks...)

	return schema
}
