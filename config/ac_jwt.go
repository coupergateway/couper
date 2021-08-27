package config

import (
	"github.com/hashicorp/hcl/v2"
)

// Internally used for 'error_handler'.
var _ Body = &JWT{}

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
	SignatureAlgorithm string   `hcl:"signature_algorithm"`

	// Internally used for 'error_handler'.
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (j *JWT) HCLBody() hcl.Body {
	return j.Remain
}
