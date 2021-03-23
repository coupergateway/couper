package config

import "github.com/hashicorp/hcl/v2"

// Inline defines the <Inline> interface.
type Inline interface {
	Body
	Schema(inline bool) *hcl.BodySchema
}

type BackendReference interface {
	Reference() string
}

type Body interface {
	HCLBody() hcl.Body
}
