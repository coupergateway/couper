package lib

import (
	"fmt"
	"net/url"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/coupergateway/couper/config"
)

const (
	FnSamlSsoURL            = "saml_sso_url"
	NameIDFormatUnspecified = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
)

// SAMLMetadataProvider abstracts metadata access for the SAML SSO URL function.
type SAMLMetadataProvider interface {
	Metadata() (*types.EntityDescriptor, error)
}

// SAMLConfigWithProvider combines SAML config with its metadata provider.
type SAMLConfigWithProvider struct {
	Config   *config.SAML
	Provider SAMLMetadataProvider
}

var NoOpSamlSsoURLFunction = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "saml_label",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
		if len(args) > 0 {
			return cty.StringVal(""), fmt.Errorf("missing saml block with referenced label %q", args[0].AsString())
		}
		return cty.StringVal(""), fmt.Errorf("missing saml definitions")
	},
})

func NewSamlSsoURLFunction(configs []SAMLConfigWithProvider, origin *url.URL) function.Function {
	samlEntities := make(map[string]SAMLConfigWithProvider)
	for _, conf := range configs {
		samlEntities[conf.Config.Name] = conf
	}

	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "saml_label",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			label := args[0].AsString()
			ent, exist := samlEntities[label]
			if !exist {
				return NoOpSamlSsoURLFunction.Call(args)
			}

			metadata, err := ent.Provider.Metadata()
			if err != nil {
				return cty.StringVal(""), fmt.Errorf("failed to get SAML metadata for %q: %w", label, err)
			}

			var ssoURL string
			for _, ssoService := range metadata.IDPSSODescriptor.SingleSignOnServices {
				if ssoService.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
					ssoURL = ssoService.Location
					continue
				}
			}

			nameIDFormat := getNameIDFormat(metadata.IDPSSODescriptor.NameIDFormats)

			absAcsURL, err := AbsoluteURL(ent.Config.SpAcsURL, origin)
			if err != nil {
				return cty.StringVal(""), err
			}

			sp := &saml2.SAMLServiceProvider{
				AssertionConsumerServiceURL: absAcsURL,
				IdentityProviderSSOURL:      ssoURL,
				ServiceProviderIssuer:       ent.Config.SpEntityID,
				SignAuthnRequests:           false,
			}
			if nameIDFormat != "" {
				sp.NameIdFormat = nameIDFormat
			}

			samlSsoURL, err := sp.BuildAuthURL("")
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(samlSsoURL), nil
		},
	})
}

func getNameIDFormat(supportedNameIDFormats []types.NameIDFormat) string {
	nameIDFormat := ""
	if isSupportedNameIDFormat(supportedNameIDFormats, NameIDFormatUnspecified) {
		nameIDFormat = NameIDFormatUnspecified
	} else if len(supportedNameIDFormats) > 0 {
		nameIDFormat = supportedNameIDFormats[0].Value
	}
	return nameIDFormat
}

func isSupportedNameIDFormat(supportedNameIDFormats []types.NameIDFormat, nameIDFormat string) bool {
	for _, n := range supportedNameIDFormats {
		if n.Value == nameIDFormat {
			return true
		}
	}
	return false
}
