package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/config/schema"
)

var (
	_ schema.BodySchema = &Websockets{}
)

type Websockets struct {
	Remain hcl.Body `hcl:",remain"`
}

// Inline implements the <Inline> interface.
func (w Websockets) Inline() interface{} {
	type Inline struct {
		meta.RequestHeadersAttributes
		meta.ResponseHeadersAttributes
		Timeout string `hcl:"timeout,optional" docs:"The total deadline [duration](#duration) a WebSocket connection has to exist."`
	}

	return &Inline{}
}

func (w Websockets) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(w)
	i, _ := gohcl.ImpliedBodySchema(w.Inline())

	return meta.MergeSchemas(s, i, meta.RequestHeadersAttributesSchema, meta.ResponseHeadersAttributesSchema)
}
