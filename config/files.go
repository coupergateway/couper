package config

type Files struct {
	AccessControl []string `hcl:"access_control,optional"`
	DocumentRoot  string   `hcl:"document_root"`
	ErrorFile     string   `hcl:"error_file,optional"` // TODO: error_${status}.html ?
}
