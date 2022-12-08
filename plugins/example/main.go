//go:generate go build -v -buildmode=plugin -o example_plugin.so -gcflags "all=-N -l" ./

package main

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/schema"
	"github.com/avenga/couper/plugins"
)

// Ensures interface compatibility
var (
	_ plugins.Config         = &Example{}
	_ plugins.ConnectionHook = &Example{}

	_ schema.BodySchema = &Example{}
)

// Plugin variable gets loaded as pointer symbol from Couper
var Plugin = Example{}

type Example struct {
	Test string `hcl:"test"`
}

func (ep *Example) Connect(ctx context.Context, args ...any) {
	//TODO implement me
	panic("implement me")
}

// Definition returns a hcl-schema which will be loaded while reading a defined parent block.
func (ep *Example) Definition() (parent plugins.MountPoint, header *hcl.BlockHeaderSchema, schema schema.BodySchema) {
	return plugins.Endpoint, &hcl.BlockHeaderSchema{
		Type:       "ldap_connector",
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
