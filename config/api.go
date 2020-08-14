package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Api struct {
	AccessControl        []string    `hcl:"access_control,optional"`
	Backend              string      `hcl:"backend,optional"`
	BasePath             string      `hcl:"base_path,optional"`
	DisableAccessControl []string    `hcl:"disable_access_control,optional"`
	Endpoint             []*Endpoint `hcl:"endpoint,block"`
	ErrorFile            string      `hcl:"error_file,optional"`
	InlineDefinition     hcl.Body    `hcl:",remain" json:"-"`
}
