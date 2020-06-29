package config

type Spa struct {
	BootstrapFile string   `hcl:"bootstrap_file"`
	Paths         []string `hcl:"paths"`
}
