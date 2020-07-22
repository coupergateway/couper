package config

type Gateway struct {
	Addr   string
	Server []*Server `hcl:"server,block"`
	WD     string
	// Defaults
}
