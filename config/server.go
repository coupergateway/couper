package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &Server{}

// Server represents the <Server> object.
type Server struct {
	AccessControl        []string    `hcl:"access_control,optional" docs:"[access controls](../access-control) to protect the server. Inherited by nested blocks."`
	APIs                 APIs        `hcl:"api,block"`
	BasePath             string      `hcl:"base_path,optional" docs:"the path prefix for all requests"`
	CORS                 *CORS       `hcl:"cors,block"`
	DisableAccessControl []string    `hcl:"disable_access_control,optional" docs:"disables access controls by name"`
	Endpoints            Endpoints   `hcl:"endpoint,block"`
	ErrorFile            string      `hcl:"error_file,optional" docs:"location of the error file template"`
	Files                FilesBlocks `hcl:"files,block"`
	Hosts                []string    `hcl:"hosts,optional" docs:""`
	Name                 string      `hcl:"name,label,optional"`
	Remain               hcl.Body    `hcl:",remain"`
	SPAs                 SPAs        `hcl:"spa,block"`
}

// Servers represents a list of <Server> objects.
type Servers []*Server

// Inline implements the <Inline> interface.
func (s Server) Inline() interface{} {
	type Inline struct {
		meta.ResponseHeadersAttributes
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (s Server) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(s)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(s.Inline())
	return meta.MergeSchemas(schema, meta.ResponseHeadersAttributesSchema, meta.LogFieldsAttributeSchema)
}
