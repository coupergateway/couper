package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &Spa{}

type SPAs []*Spa

// Spa represents the <Spa> object.
type Spa struct {
	AccessControl        []string `hcl:"access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	BootstrapFile        string   `hcl:"bootstrap_file"`
	CORS                 *CORS    `hcl:"cors,block"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	Name                 string   `hcl:"name,label,optional"`
	Paths                []string `hcl:"paths"`
	Remain               hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (s Spa) HCLBody() hcl.Body {
	return s.Remain
}

// Inline implements the <Inline> interface.
func (s Spa) Inline() interface{} {
	type Inline struct {
		meta.ResponseHeadersAttributes
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (s Spa) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(s)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(s.Inline())
	return meta.MergeSchemas(schema, meta.ResponseHeadersAttributesSchema, meta.LogFieldsAttributeSchema)
}
