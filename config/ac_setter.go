package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var ACSetterSchema, _ = gohcl.ImpliedBodySchema(&AccessControlSetter{})

type AccessControlSetter struct {
	ErrorHandler []*ErrorHandler `hcl:"error_handler,block"`
}

func (acs *AccessControlSetter) Set(ehConf *ErrorHandler) {
	acs.ErrorHandler = append(acs.ErrorHandler, ehConf)
}

func SchemaWithACSetter(schema *hcl.BodySchema) *hcl.BodySchema {
	schema.Attributes = append(schema.Attributes, ACSetterSchema.Attributes...)
	schema.Blocks = append(schema.Blocks, ACSetterSchema.Blocks...)

	return schema
}
