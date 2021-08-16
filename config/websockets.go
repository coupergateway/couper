package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ Inline = &Websockets{}

	WebsocketsInlineSchema = Websockets{}.Schema(true)
)

type Websockets struct {
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (w Websockets) HCLBody() hcl.Body {
	return w.Remain
}

// Schema implements the <Inline> interface.
func (w Websockets) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(w)
	if !inline {
		return schema
	}

	type Inline struct {
		SetRequestHeaders map[string]string `hcl:"set_request_headers,optional"`
		Timeout           string            `hcl:"timeout,optional"`
	}

	schema, _ = gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
