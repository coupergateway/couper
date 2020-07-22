package config

type Files struct {
	BasePath     string `hcl:"base_path,optional"`
	DocumentRoot string `hcl:"document_root"`
	ErrorFile    string `hcl:"error_file,optional"` // TODO: error_${status}.html ?
}
