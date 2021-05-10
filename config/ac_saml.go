package config

import "github.com/hashicorp/hcl/v2"

var _ Body = &SAML{}

// SAML represents the <SAML> object.
type SAML struct {
	AccessControlSetter
	ArrayAttributes []string `hcl:"array_attributes,optional"`
	IdpMetadataFile string   `hcl:"idp_metadata_file"`
	Name            string   `hcl:"name,label"`
	Remain          hcl.Body `hcl:",remain"`
	SpAcsUrl        string   `hcl:"sp_acs_url"`
	SpEntityId      string   `hcl:"sp_entity_id"`
}

// HCLBody implements the <Body> interface.
func (s *SAML) HCLBody() hcl.Body {
	return s.Remain
}
