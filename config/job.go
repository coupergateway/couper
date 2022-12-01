package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var (
	_ Inline = &Job{}
)

// Job represents the <Job> object.
type Job struct {
	Interval string   `hcl:"interval" docs:"Execution interval" type:"duration"`
	Name     string   `hcl:"name,label"`
	Remain   hcl.Body `hcl:",remain"`
	Requests Requests `hcl:"request,block" docs:"Configures a [request](/configuration/block/request)."`

	// Internally used
	Endpoint *Endpoint
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
