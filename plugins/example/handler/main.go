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
	Name string `hcl:"name,label"`
}

type container struct {
	Examples []*Example `hcl:"my_access_control,block"`
}

// Definition writes multiple hcl-schema to the given channel which will be loaded while reading a defined parent block.
func (ep *Example) Definition(ch chan<- plugins.SchemaDefinition) {
	ch <- plugins.SchemaDefinition{
		Parent: plugins.Definitions,
		BlockHeader: &hcl.BlockHeaderSchema{
			Type:       "my_access_control",
			LabelNames: []string{"name"},
		},
		Body: ep,
	}
}

func (ep *Example) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(ep)
	return s
}

func (ep *Example) Decode(f func(ref any) error) error {
	c := &container{}
	return f(c) // TODO: manage container content
}
