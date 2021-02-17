package config

import "github.com/hashicorp/hcl/v2"

// Inline defines the <Inline> interface.
type Inline interface {
	HCLBody() hcl.Body
	Reference() string
	Schema(inline bool) *hcl.BodySchema
}
