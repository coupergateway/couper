package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Endpoint struct {
	AccessControl        []string `hcl:"access_control,optional"`
	Backend              string   `hcl:"backend,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	InlineDefinition     hcl.Body `hcl:",remain" json:"-"`
	Path                 string   `hcl:"path,optional"`
	Pattern              string   `hcl:"path,label"`
}
