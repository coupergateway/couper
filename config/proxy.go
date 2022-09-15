package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference = &Proxy{}
	_ Body             = &Proxy{}
	_ Inline           = &Proxy{}
)

// Proxy represents the <Proxy> object.
type Proxy struct {
	BackendName string   `hcl:"backend,optional" docs:"backend block reference"`
	Name        string   `hcl:"name,label,optional"`
	Remain      hcl.Body `hcl:",remain"`
	Websockets  *bool    `hcl:"websockets,optional" docs:"Allows support for WebSockets. This attribute is only allowed in the \"default\" proxy block. Other {proxy} blocks, {request} blocks or {response} blocks are not allowed within the current {endpoint} block."`

	// internally used
	Backend *hclsyntax.Body
}

// Proxies represents a list of <Proxy> objects.
type Proxies []*Proxy

// Reference implements the <BackendReference> interface.
func (p Proxy) Reference() string {
	return p.BackendName
}

// HCLBody implements the <Body> interface.
func (p Proxy) HCLBody() hcl.Body {
	return p.Remain
}

// Inline implements the <Inline> interface.
func (p Proxy) Inline() interface{} {
	type Inline struct {
		meta.RequestHeadersAttributes
		meta.ResponseHeadersAttributes
		meta.FormParamsAttributes
		meta.QueryParamsAttributes
		Backend        *Backend    `hcl:"backend,block"`
		ExpectedStatus []int       `hcl:"expected_status,optional" docs:"If defined, the response status code will be verified against this list of codes. If the status code not included in this list an {unexpected_status} error will be thrown which can be handled with an [{error_handler}](error_handler)."`
		URL            string      `hcl:"url,optional" docs:"If defined, the host part of the URL must be the same as the {origin} attribute of the corresponding backend."`
		Websockets     *Websockets `hcl:"websockets,block"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (p Proxy) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(p)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(p.Inline())
	backup := schema.Blocks[:]
	schema.Blocks = nil

	if p.BackendName == "" {
		var blocks []hcl.BlockHeaderSchema

		for _, block := range backup {
			if block.Type == "backend" {
				blocks = append(blocks, block)
			}
		}

		schema.Blocks = blocks
	}

	if p.Websockets == nil {
		for _, block := range backup {
			if block.Type == "websockets" {
				// No websockets flag is set, websocket block is allowed.
				schema.Blocks = append(schema.Blocks, block)
			}
		}
	}

	return meta.MergeSchemas(schema, meta.ModifierAttributesSchema)
}
