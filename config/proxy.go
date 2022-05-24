package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference = &Proxy{}
	_ Inline           = &Proxy{}
)

// Proxy represents the <Proxy> object.
type Proxy struct {
	BackendName string   `hcl:"backend,optional"`
	Name        string   `hcl:"name,label,optional"`
	Remain      hcl.Body `hcl:",remain"`
	Websockets  *bool    `hcl:"websockets,optional"`

	// internally used
	Backend hcl.Body
}

// Proxies represents a list of <Proxy> objects.
type Proxies []*Proxy

// Reference implements the <BackendReference> interface.
func (p Proxy) Reference() string {
	return p.BackendName
}

// HCLBody implements the <Inline> interface.
func (p Proxy) HCLBody() hcl.Body {
	return p.Remain
}

// Inline implements the <Inline> interface.
func (p Proxy) Inline() interface{} {
	type Inline struct {
		meta.Attributes
		Backend        *Backend    `hcl:"backend,block"`
		ExpectedStatus []int       `hcl:"expected_status,optional"`
		URL            string      `hcl:"url,optional"`
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

	return meta.SchemaWithAttributes(schema)
}
