package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var (
	_ Inline = &Websockets{}

	WebsocketsInlineSchema = Websockets{}.Schema(true)
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

// Schema implements the <Inline> interface.
func (w Websockets) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(w)
	if !inline {
		return schema
	}

	schema, _ = gohcl.ImpliedBodySchema(w.Inline())

	return meta.MergeSchemas(schema, meta.RequestHeadersAttributesSchema, meta.ResponseHeadersAttributesSchema)
}
