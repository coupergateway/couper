package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Path struct {
	Server  *Server  `hcl:"-"` // parent
	Pattern string   `hcl:"path,label"`
	Backend string   `hcl:"backend,optional"`
	Options hcl.Body `hcl:",remain" json:"-"`
}

func (p *Path) String() string {
	return p.Server.Name + ": " + p.Pattern
}
