package saml

type SAML struct {
	ArrayAttributes []string `hcl:"array_attributes,optional"`
	IdpMetadataFile string   `hcl:"idp_metadata_file"`
	Name            string   `hcl:"name,label"`
	SpAcsUrl        string   `hcl:"sp_acs_url"`
	SpEntityId      string   `hcl:"sp_entity_id"`
}
