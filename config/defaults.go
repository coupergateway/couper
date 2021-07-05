package config

type Defaults struct {
	EnvironmentVariables map[string]string `hcl:"environment_variables,optional"`
}
