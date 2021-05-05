package config

import "github.com/hashicorp/hcl/v2"

// Inline defines the <Inline> interface.
type Inline interface {
	Body
	Schema(inline bool) *hcl.BodySchema
}

// BackendReference defines the <BackendReference> interface.
type BackendReference interface {
	Reference() string
}

// Body defines the <Body> interface.
type Body interface {
	HCLBody() hcl.Body
}
