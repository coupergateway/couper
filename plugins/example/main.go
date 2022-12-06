//go:generate go build -v -buildmode=plugin -o example_plugin.so -gcflags "all=-N -l" ./

package main

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/plugins"
)

// Ensures interface compatibility
var (
	_ plugins.Config        = &Example{}
	_ plugins.RoundtripHook = &Example{}
)

// Plugin variable gets loaded as pointer symbol from Couper
var Plugin = Example{}

type Example struct{}

// Register returns a hcl-schema which will be loaded while reading a defined parent block.
func (ep *Example) Register() (parent string, header *hcl.BlockHeaderSchema, schema *hcl.BodySchema) {
	return "definitions", &hcl.BlockHeaderSchema{
			Type:          "poc",
			LabelNames:    nil,
			LabelOptional: false,
		}, &hcl.BodySchema{
			Attributes: []hcl.AttributeSchema{
				{
					Name:     "test",
					Required: false,
				},
			},
		}
}

func (ep *Example) RegisterRoundtripFunc(kind plugins.HookKind, tripper http.RoundTripper) {
	//TODO implement me
	panic("implement me")
}
