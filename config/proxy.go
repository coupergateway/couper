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
	Name        string   `hcl:"name,label"`
	Remain      hcl.Body `hcl:",remain"`
	// internally used
	Backend hcl.Body
}

// Reference implements the <Inline> interface.
func (p Proxy) Reference() string {
	return p.BackendName
}

// Proxies represents a list of <Proxy> objects.
type Proxies []*Proxy

// HCLBody implements the <Inline> interface.
func (p Proxy) HCLBody() hcl.Body {
	return p.Remain
}

// Schema implements the <Inline> interface.
func (p Proxy) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(p)
		return schema
	}

	type Inline struct {
		meta.Attributes
		Backend *Backend `hcl:"backend,block"`
		URL     string   `hcl:"url,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if p.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, p.HCLBody())
}
