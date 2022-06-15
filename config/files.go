package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Files{}

type FilesBlocks []*Files

// Files represents the <Files> object.
type Files struct {
	AccessControl        []string `hcl:"access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	CORS                 *CORS    `hcl:"cors,block"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	DocumentRoot         string   `hcl:"document_root"`
	ErrorFile            string   `hcl:"error_file,optional"`
	Name                 string   `hcl:"name,label,optional"`
	Remain               hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (f Files) HCLBody() hcl.Body {
	return f.Remain
}

// Inline implements the <Inline> interface.
func (f Files) Inline() interface{} {
	type Inline struct {
		AddResponseHeaders map[string]string         `hcl:"add_response_headers,optional"`
		DelResponseHeaders []string                  `hcl:"remove_response_headers,optional"`
		SetResponseHeaders map[string]string         `hcl:"set_response_headers,optional"`
		LogFields          map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
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

	return schema
}
