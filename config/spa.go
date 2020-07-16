package config

type Spa struct {
	BasePath      string   `hcl:"base_path,optional"`
	BootstrapFile string   `hcl:"bootstrap_file"`
	Paths         []string `hcl:"paths"`
}
