package config

import (
	"github.com/hashicorp/hcl/v2"
)

var _ Body = &SAML{}

// Claims represents the <Claims> object.
type Claims hcl.Expression

// JWT represents the <JWT> object.
type JWT struct {
	AccessControlSetter
	Claims             Claims   `hcl:"claims,optional"`
	ClaimsRequired     []string `hcl:"required_claims,optional"`
	Cookie             string   `hcl:"cookie,optional"`
	Header             string   `hcl:"header,optional"`
	Key                string   `hcl:"key,optional"`
	KeyFile            string   `hcl:"key_file,optional"`
	Name               string   `hcl:"name,label"`
	PostParam          string   `hcl:"post_param,optional"`
	QueryParam         string   `hcl:"query_param,optional"`
	Remain             hcl.Body `hcl:",remain"`
	SignatureAlgorithm string   `hcl:"signature_algorithm"`
}

// HCLBody implements the <Body> interface.
func (j *JWT) HCLBody() hcl.Body {
	return j.Remain
}
