package config

import "github.com/hashicorp/hcl/v2"

type DefaultEnvVars map[string]string

type Defaults struct {
	EnvironmentVariables DefaultEnvVars `hcl:"environment_variables,optional"`
}

type DefaultsBlock struct {
	Defaults *Defaults `hcl:"defaults,block"`
	Remain   hcl.Body  `hcl:",remain"`
}
