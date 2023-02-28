package config

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

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
