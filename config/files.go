package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &Files{}

type FilesBlocks []*Files

// Files represents the <Files> object.
type Files struct {
	AccessControl        []string `hcl:"access_control,optional" docs:"Sets predefined access control for this block context."`
	BasePath             string   `hcl:"base_path,optional" docs:"Configures the path prefix for all requests."`
	CORS                 *CORS    `hcl:"cors,block"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	DocumentRoot         string   `hcl:"document_root" docs:"Location of the document root (directory)."`
	ErrorFile            string   `hcl:"error_file,optional" docs:"Location of the error file template."`
	Name                 string   `hcl:"name,label,optional"`
	Remain               hcl.Body `hcl:",remain"`
}

// Inline implements the <Inline> interface.
func (f Files) Inline() interface{} {
	type Inline struct {
		meta.ResponseHeadersAttributes
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (f Files) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(f)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(f.Inline())
	return meta.MergeSchemas(schema, meta.ResponseHeadersAttributesSchema, meta.LogFieldsAttributeSchema)
}
