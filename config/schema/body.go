package schema

import "github.com/hashicorp/hcl/v2"

type BodySchema interface {
	Schema() *hcl.BodySchema
}
