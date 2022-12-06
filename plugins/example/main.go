//go:generate go build -v -buildmode=plugin -o example_plugin.so -gcflags "all=-N -l" ./

package main

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/schema"
	"github.com/avenga/couper/plugins"
)

// Ensures interface compatibility
var (
	_ plugins.Config        = &Example{}
	_ plugins.RoundtripHook = &Example{}

	_ schema.BodySchema = &Example{}
)

// Plugin variable gets loaded as pointer symbol from Couper
var Plugin = Example{}

type Example struct {
	Test string `hcl:"test"`
}

func (ep *Example) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(ep)
	return s
}

// Definition returns a hcl-schema which will be loaded while reading a defined parent block.
func (ep *Example) Definition() (parent string, header *hcl.BlockHeaderSchema, schema schema.BodySchema) {
	// TODO: Couper validate step
	return "definitions", &hcl.BlockHeaderSchema{
		Type:       "poc",
		LabelNames: []string{"name"},
	}, ep
}

func (ep *Example) RegisterRoundtripFunc(kind plugins.HookKind, rt http.RoundTripper) {
	//TODO implement me
	panic("implement me")
}
