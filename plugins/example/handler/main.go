//go:generate go build -v -buildmode=plugin -o example.so -gcflags "all=-N -l" ./

package main

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/schema"
	"github.com/avenga/couper/plugins"
)

// Ensures interface compatibility
var (
	_ plugins.Config = &Example{}
	//_ plugins.HandlerHook = &Example{}

	_ schema.BodySchema = &Example{}
)

// Plugin variable gets loaded as pointer symbol from Couper
var Plugin = Example{}

type Example struct {
	Test string `hcl:"test"`
}

// Definition returns a hcl-schema which will be loaded while reading a defined parent block.
func (ep *Example) Definition() (parent plugins.MountPoint, header *hcl.BlockHeaderSchema, schema schema.BodySchema) {
	return plugins.Definitions, &hcl.BlockHeaderSchema{
		Type:       "my_access_control",
		LabelNames: []string{"name"},
	}, ep
}

func (ep *Example) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(ep)
	return s
}

func (ep *Example) Validate(ctx *hcl.EvalContext, body hcl.Body) {
	//TODO implement me
	panic("implement me")
}
