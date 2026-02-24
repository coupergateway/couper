package config

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/meta"
	"github.com/coupergateway/couper/errors"
)

var (
	_ BackendInitialization = &SAML{}
	_ BackendReference      = &SAML{}
	_ Body                  = &SAML{}
	_ Inline                = &SAML{}
)

// SAML represents the <SAML> object.
type SAML struct {
	ErrorHandlerSetter
	ArrayAttributes  []string `hcl:"array_attributes,optional" docs:"A list of assertion attributes that may have several values. Results in at least an empty array in {request.context.<label>.attributes.<name>}"`
	IdpMetadataFile  string   `hcl:"idp_metadata_file,optional" docs:"File reference to the Identity Provider metadata XML file. Mutually exclusive with {idp_metadata_url}."`
	IdpMetadataURL   string   `hcl:"idp_metadata_url,optional" docs:"URL to fetch the Identity Provider metadata XML. Mutually exclusive with {idp_metadata_file}."`
	MetadataTTL      string   `hcl:"metadata_ttl,optional" docs:"Time period the IdP metadata stays valid and may be cached." type:"duration" default:"1h"`
	MetadataMaxStale string   `hcl:"metadata_max_stale,optional" docs:"Time period the cached IdP metadata stays valid after its TTL has passed." type:"duration" default:"1h"`
	BackendName      string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for IdP metadata requests. Mutually exclusive with {backend} block."`
	Name             string   `hcl:"name,label"`
	Remain           hcl.Body `hcl:",remain"`
	SpAcsURL         string   `hcl:"sp_acs_url" docs:"The URL of the Service Provider's ACS endpoint. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [{accept_forwarded_url} attribute](settings) if Couper is running behind a proxy."`
	SpEntityID       string   `hcl:"sp_entity_id" docs:"The Service Provider's entity ID."`

	// internally used
	MetadataBytes []byte
	Backend       *hclsyntax.Body
}

// Prepare implements the BackendInitialization interface.
func (s *SAML) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if s.IdpMetadataURL != "" {
		s.Backend, err = backendFunc("idp_metadata_url", s.IdpMetadataURL, s)
		if err != nil {
			return err
		}
	}

	if err = s.check(); err != nil {
		return errors.Configuration.Label(s.Name).With(err)
	}
	return nil
}

// Reference implements the BackendReference interface.
func (s *SAML) Reference() string {
	return s.BackendName
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (s *SAML) HCLBody() *hclsyntax.Body {
	return s.Remain.(*hclsyntax.Body)
}

func (s *SAML) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend *Backend `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for IdP metadata requests. Mutually exclusive with {backend} attribute."`
	}

	return &Inline{}
}

func (s *SAML) check() error {
	if s.IdpMetadataFile == "" && s.IdpMetadataURL == "" {
		return fmt.Errorf("one of idp_metadata_file or idp_metadata_url is required")
	}

	if s.IdpMetadataFile != "" && s.IdpMetadataURL != "" {
		return fmt.Errorf("idp_metadata_file and idp_metadata_url are mutually exclusive")
	}

	if s.IdpMetadataFile != "" {
		if s.BackendName != "" || s.Backend != nil {
			return fmt.Errorf("backend is only valid with idp_metadata_url")
		}
		if s.MetadataTTL != "" {
			return fmt.Errorf("metadata_ttl is only valid with idp_metadata_url")
		}
		if s.MetadataMaxStale != "" {
			return fmt.Errorf("metadata_max_stale is only valid with idp_metadata_url")
		}
	}

	return nil
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
