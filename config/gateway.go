package config

type Gateway struct {
	Addr        string
	Server      []*Server    `hcl:"server,block"`
	Definitions *Definitions `hcl:"definitions,block"`
}
