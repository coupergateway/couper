package config

import "github.com/hashicorp/hcl/v2"

// Inline defines the <Inline> interface.
type Inline interface {
	Body
	Inline() interface{}
	Schema(inline bool) *hcl.BodySchema
}

// BackendReference defines the <BackendReference> interface.
type BackendReference interface {
	Reference() string
}

type PrepareBackendFunc func(string, string, Inline) (hcl.Body, error)

type BackendInitialization interface {
	Prepare(backendFunc PrepareBackendFunc) error
}

// Body defines the <Body> interface.
type Body interface {
	HCLBody() hcl.Body
}
