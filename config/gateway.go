package config

type Gateway struct {
	Applications []*Application `hcl:"application,block"`
	// Defaults
}
