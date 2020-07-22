package config

type Spa struct {
	AccessControl        []string `hcl:"access_control,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	BootstrapFile        string   `hcl:"bootstrap_file"`
	Paths                []string `hcl:"paths"`
}
