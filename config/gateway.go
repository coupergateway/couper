package config

type Gateway struct {
	Frontends []*Frontend `hcl:"frontend,block"`
	// Defaults
}
