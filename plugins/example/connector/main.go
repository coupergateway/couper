//go:generate go build -v -buildmode=plugin -o example.so -gcflags "all=-N -l" ./

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

type container struct {
	Examples []*Example `hcl:"my_connector,block"`
}

func (ep *Example) Connect(ctx context.Context, args ...any) {
	//TODO implement me
	panic("implement me")
}

// Definition writes multiple hcl-schema to the given channel which will be loaded while reading a defined parent block.
func (ep *Example) Definition(ch chan<- plugins.SchemaDefinition) {
	ch <- plugins.SchemaDefinition{
		Parent: plugins.Endpoint,
		BlockHeader: &hcl.BlockHeaderSchema{
			Type:       "my_connector",
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
