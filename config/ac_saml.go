package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ Body = &SAML{}
	//_ Inline = &SAML{}
)

// SAML represents the <SAML> object.
type SAML struct {
	ErrorHandlerSetter
	ArrayAttributes []string `hcl:"array_attributes,optional" docs:"A list of assertion attributes that may have several values. Results in at least an empty array in {request.context.<label>.attributes.<name>}"`
	IdpMetadataFile string   `hcl:"idp_metadata_file" docs:"File reference to the Identity Provider metadata XML file."`
	Name            string   `hcl:"name,label"`
	Remain          hcl.Body `hcl:",remain"`
	SpAcsURL        string   `hcl:"sp_acs_url" docs:"The URL of the Service Provider's ACS endpoint. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [{accept_forwarded_url} attribute](settings) if Couper is running behind a proxy."`
	SpEntityID      string   `hcl:"sp_entity_id" docs:"The Service Provider's entity ID."`

	// internally used
	MetadataBytes []byte
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (s *SAML) HCLBody() *hclsyntax.Body {
	return s.Remain.(*hclsyntax.Body)
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
