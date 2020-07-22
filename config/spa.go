package config

type Spa struct {
	AccessControl
	BasePath      string   `hcl:"base_path,optional"`
	BootstrapFile string   `hcl:"bootstrap_file"`
	Paths         []string `hcl:"paths"`
}
