package config

import "github.com/hashicorp/hcl/v2"

const (
	WildcardCtxKey = "route_wildcard"
)

type Gateway struct {
	Context     *hcl.EvalContext `hcl:"-"`
	Definitions *Definitions     `hcl:"definitions,block"`
	Server      []*Server        `hcl:"server,block"`
}
