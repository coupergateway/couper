package config

type Server struct {
	AccessControl []string `hcl:"access_control,optional"`
	API           *Api     `hcl:"api,block"`
	BasePath      string   `hcl:"base_path,optional"`
	Domains       []string `hcl:"domains,optional"`
	Files         *Files   `hcl:"files,block"`
	Name          string   `hcl:"name,label"`
	Spa           *Spa     `hcl:"spa,block"`
}
