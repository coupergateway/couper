package config

// Spa represents the <Spa> object.
type Spa struct {
	AccessControl        []string `hcl:"access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	BootstrapFile        string   `hcl:"bootstrap_file"`
	CORS                 *CORS    `hcl:"cors,block"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	Paths                []string `hcl:"paths"`
}
