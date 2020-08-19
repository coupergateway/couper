package config

type Server struct {
	AccessControl        []string `hcl:"access_control,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	API                  *Api     `hcl:"api,block"`
	BasePath             string   `hcl:"base_path,optional"`
	Listen               []string `hcl:"listen,optional"`
	Files                *Files   `hcl:"files,block"`
	Name                 string   `hcl:"name,label"`
	Spa                  *Spa     `hcl:"spa,block"`
}
