package config

// Files represents the <Files> object.
type Files struct {
	AccessControl        []string `hcl:"access_control,optional"`
	BasePath             string   `hcl:"base_path,optional"`
	CORS                 *CORS    `hcl:"cors,block"`
	DisableAccessControl []string `hcl:"disable_access_control,optional"`
	DocumentRoot         string   `hcl:"document_root"`
	ErrorFile            string   `hcl:"error_file,optional"`
}
