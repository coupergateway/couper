package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Endpoint struct {
	Server  *Server  `hcl:"-"` // parent
	Pattern string   `hcl:"path,label"`
	Backend string   `hcl:"backend,optional"`
	Options hcl.Body `hcl:",remain" json:"-"`
}

func (e *Endpoint) String() string {
	return e.Server.Name + ": " + e.Pattern
}
