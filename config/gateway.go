package config

import "github.com/hashicorp/hcl/v2"

const (
	WildcardCtxKey = "route_wildcard"
)

type Gateway struct {
	Addr        string
	Context     *hcl.EvalContext
	Definitions *Definitions `hcl:"definitions,block"`
	Server      []*Server    `hcl:"server,block"`
	WorkDir     string
}
