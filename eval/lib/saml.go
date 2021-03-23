package lib

import (
	"encoding/xml"
	"io/ioutil"
	"path/filepath"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/avenga/couper/config"
)

const FnSamlSsoUrl = "saml_sso_url"

func NewSamlSsoUrlFunction(samlConfigs []*config.SAML) function.Function {
	samls := make(map[string]*config.SAML)
	for _, s := range samlConfigs {
		samls[s.Name] = s
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
			saml := samls[label]
			p, err := filepath.Abs(saml.IdpMetadataFile)
			if err != nil {
				return cty.StringVal(""), err
			}

			rawMetadata, err := ioutil.ReadFile(p)
			if err != nil {
				return cty.StringVal(""), err
			}

			metadata := &types.EntityDescriptor{}
			err = xml.Unmarshal(rawMetadata, metadata)
			if err != nil {
				return cty.StringVal(""), err
			}

			var ssoUrl string
			for _, ssoService := range metadata.IDPSSODescriptor.SingleSignOnServices {
				if ssoService.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
					ssoUrl = ssoService.Location
					continue
				}
			}

			nameIDFormat := ""
			if len(metadata.IDPSSODescriptor.NameIDFormats) > 0 {
				nameIDFormat = metadata.IDPSSODescriptor.NameIDFormats[0].Value
			}

			sp := &saml2.SAMLServiceProvider{
				AssertionConsumerServiceURL: saml.SpAcsUrl,
				IdentityProviderSSOURL:      ssoUrl,
				ServiceProviderIssuer:       saml.SpEntityId,
				SignAuthnRequests:           false,
			}
			if nameIDFormat != "" {
				sp.NameIdFormat = nameIDFormat
			}

			samlSsoUrl, err := sp.BuildAuthURL("")
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(samlSsoUrl), nil
		},
	})
}
