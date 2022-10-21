package config

import (
	"github.com/avenga/couper/config/meta"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var (
	_ Body   = &Job{}
	_ Inline = &Job{}
)

// Job represents the <Job> object.
type Job struct {
	Interval string   `hcl:"interval" docs:"Execution interval"`
	Name     string   `hcl:"name,label"`
	Remain   hcl.Body `hcl:",remain"`
	Requests Requests `hcl:"request,block" docs:"[{request}](request) block definition."`

	// Internally used
	Endpoint *Endpoint
}

// HCLBody implements the <Body> interface.
func (j Job) HCLBody() *hclsyntax.Body {
	return j.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (j Job) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (j Job) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(j)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(j.Inline())

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}
