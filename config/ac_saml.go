package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &SAML{}

// SAML represents the <SAML> object.
type SAML struct {
	ErrorHandlerSetter
	ArrayAttributes []string `hcl:"array_attributes,optional"`
	IdpMetadataFile string   `hcl:"idp_metadata_file"`
	Name            string   `hcl:"name,label"`
	Remain          hcl.Body `hcl:",remain"`
	SpAcsUrl        string   `hcl:"sp_acs_url"`
	SpEntityId      string   `hcl:"sp_entity_id"`

	// internally used
	MetadataBytes []byte
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (s *SAML) HCLBody() hcl.Body {
	return s.Remain
}

func (s *SAML) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (s *SAML) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(s)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(s.Inline())
	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}
