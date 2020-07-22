package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Endpoint struct {
	AccessControl
	Backend string   `hcl:"backend,optional"`
	Options hcl.Body `hcl:",remain" json:"-"`
	Pattern string   `hcl:"path,label"`
	Server  *Server  `hcl:"-"` // parent
}

func (e *Endpoint) String() string {
	return e.Server.Name + ": " + e.Pattern
}
