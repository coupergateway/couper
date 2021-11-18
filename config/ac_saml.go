package config

import "github.com/hashicorp/hcl/v2"

// Internally used for 'error_handler'.
var _ Body = &SAML{}

// SAML represents the <SAML> object.
type SAML struct {
	ErrorHandlerSetter
	ArrayAttributes []string `hcl:"array_attributes,optional"`
	IdpMetadataFile string   `hcl:"idp_metadata_file"`
	Name            string   `hcl:"name,label"`
	SpAcsUrl        string   `hcl:"sp_acs_url"`
	SpEntityId      string   `hcl:"sp_entity_id"`

	// internally used
	MetadataBytes []byte

	// Internally used for 'error_handler'.
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (s *SAML) HCLBody() hcl.Body {
	return s.Remain
}
