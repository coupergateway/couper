package config

import "github.com/hashicorp/hcl/v2"

const (
	WildcardCtxKey = "route_wildcard"
)

type Hosts map[string]*Server
type Ports map[string]Hosts

type Gateway struct {
	Context     *hcl.EvalContext
	Definitions *Definitions `hcl:"definitions,block"`
	ListenPort  int
	Lookups     Ports     // map[<port:string>][<host:string>]*Server
	Server      []*Server `hcl:"server,block"`
	WorkDir     string
}
