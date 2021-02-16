package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &API{}

// API represents the <API> object.
type API struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	CORS                 *CORS     `hcl:"cors,block"`
	Backend              string    `hcl:"backend,optional"`
	BasePath             string    `hcl:"base_path,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Remain               hcl.Body  `hcl:",remain"`
}

// APIs represents a list of <API> objects.
type APIs []*API

// HCLBody implements the <Inline> interface.
func (a API) HCLBody() hcl.Body {
	return a.Remain
}

// Reference implements the <Inline> interface.
func (a API) Reference() string {
	return a.Backend
}

// Schema implements the <Inline> interface.
func (a API) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(a)
		return schema
	}

	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// The API contains a backend reference, backend block is not allowed.
	if a.Backend != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, a.HCLBody())
}
