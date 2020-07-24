package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Endpoint struct {
	AccessControl        []string `hcl:"access_control,optional"`
	Backend              string   `hcl:"backend,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	InlineDefinition     hcl.Body `hcl:",remain" json:"-"`
	Pattern              string   `hcl:"path,label"`
	Server               *Server  `hcl:"-"` // parent
}

func (e *Endpoint) String() string {
	return e.Server.Name + ": " + e.Pattern
}
