package config

type Files struct {
	AccessControl        []string `hcl:"access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	DocumentRoot         string   `hcl:"document_root"`
	ErrorFile            string   `hcl:"error_file,optional"`
}
