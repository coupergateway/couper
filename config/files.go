package config

type Files struct {
	DocumentRoot string `hcl:"document_root"`
	ErrorFile    string `hcl:"error_file,optional"` // TODO: error_${status}.html ?
}
