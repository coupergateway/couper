package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/avenga/couper/config"
)

const (
	FnSamlSsoUrl            = "saml_sso_url"
	NameIdFormatUnspecified = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
)

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
			if len(samls) == 0 {
				return cty.StringVal(""), fmt.Errorf("no saml definitions found")
			}

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

			nameIDFormat := getNameIDFormat(metadata.IDPSSODescriptor.NameIDFormats)

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

func getNameIDFormat(supportedNameIDFormats []types.NameIDFormat) string {
	nameIDFormat := ""
	if isSupportedNameIDFormat(supportedNameIDFormats, NameIdFormatUnspecified) {
		nameIDFormat = NameIdFormatUnspecified
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
