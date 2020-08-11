package config

const (
	WildcardCtxKey = "route_wildcard"
)

type Gateway struct {
	Addr        string
	Server      []*Server    `hcl:"server,block"`
	Definitions *Definitions `hcl:"definitions,block"`
	WorkDir     string
}
