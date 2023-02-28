package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/schema"
)

var _ schema.BodySchema = Defaults{}

type DefaultEnvVars map[string]string

type Defaults struct {
	EnvironmentVariables DefaultEnvVars `hcl:"environment_variables,optional" docs:"One or more environment variable assignments. Keys must be either identifiers or simple string expressions."`
}

func (d Defaults) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(d)
	return s
}

type DefaultsBlock struct {
	Defaults *Defaults `hcl:"defaults,block"`
	Remain   hcl.Body  `hcl:",remain"`
}
