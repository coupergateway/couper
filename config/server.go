package config

type Server struct {
	AccessControl        []string `hcl:"access_control,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	API                  *Api     `hcl:"api,block"`
	BasePath             string   `hcl:"base_path,optional"`
	Files                *Files   `hcl:"files,block"`
	Hosts                []string `hcl:"hosts,optional"`
	Name                 string   `hcl:"name,label"`
	Spa                  *Spa     `hcl:"spa,block"`
	Mux                  *Mux
}
