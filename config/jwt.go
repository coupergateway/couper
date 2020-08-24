package config

type Claims map[string]string

type JWT struct {
	Name               string `hcl:"name,label"`
	Cookie             string `hcl:"cookie,optional"`
	Header             string `hcl:"header,optional"`
	PostParam          string `hcl:"post_param,optional"`
	QueryParam         string `hcl:"query_param,optional"`
	Key                string `hcl:"key,optional"`
	KeyFile            string `hcl:"key_file,optional"`
	SignatureAlgorithm string `hcl:"signature_algorithm"`
	Claims             Claims `hcl:"claims,optional"`
}
