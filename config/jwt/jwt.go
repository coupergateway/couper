package jwt

import (
	"github.com/hashicorp/hcl/v2"
)

type Claims hcl.Expression

type JWT struct {
	Claims             Claims   `hcl:"claims,optional"`
	ClaimsRequired     []string `hcl:"required_claims,optional"`
	Cookie             string   `hcl:"cookie,optional"`
	Header             string   `hcl:"header,optional"`
	Key                string   `hcl:"key,optional"`
	KeyFile            string   `hcl:"key_file,optional"`
	Name               string   `hcl:"name,label"`
	PostParam          string   `hcl:"post_param,optional"`
	QueryParam         string   `hcl:"query_param,optional"`
	SignatureAlgorithm string   `hcl:"signature_algorithm"`
}
