package config

import "github.com/hashicorp/hcl/v2"

const (
	WildcardCtxKey = "route_wildcard"
)

type Domains map[string]*Server
type Ports map[string]Domains

type Gateway struct {
	Context     *hcl.EvalContext
	Definitions *Definitions `hcl:"definitions,block"`
	ListenPort  int
	Lookups     Ports     // map[<port:string>][<domain:string>]*Server
	Server      []*Server `hcl:"server,block"`
	WorkDir     string
}
