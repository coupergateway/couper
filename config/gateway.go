package config

type Gateway struct {
	Server []*Server `hcl:"server,block"`
	// Defaults
}
