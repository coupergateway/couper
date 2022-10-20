package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Inline defines the <Inline> interface.
type Inline interface {
	Inline() interface{}
	Schema(inline bool) *hcl.BodySchema
}

// BackendReference defines the <BackendReference> interface.
type BackendReference interface {
	Reference() string
}

type PrepareBackendFunc func(attr string, val string, body Body) (*hclsyntax.Body, error)

type BackendInitialization interface {
	Prepare(backendFunc PrepareBackendFunc) error
}

// Body defines the <Body> interface.
type Body interface {
	HCLBody() *hclsyntax.Body
}
