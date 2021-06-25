package lib

import (
	"encoding/xml"
	"fmt"

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

func NewSamlSsoUrlFunction(configs []*config.SAML) function.Function {
	type entity struct {
		config     *config.SAML
		descriptor *types.EntityDescriptor
		err        error
	}

	samlEntities := make(map[string]*entity)
	for _, conf := range configs {
		metadata := &types.EntityDescriptor{}
		err := xml.Unmarshal(conf.MetadataBytes, metadata)
		samlEntities[conf.Name] = &entity{
			config:     conf,
			descriptor: metadata,
			err:        err,
		}
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
			if len(samlEntities) == 0 {
				return cty.StringVal(""), fmt.Errorf("missing saml2 definitions")
			}

			if len(args) == 0 {
				return cty.StringVal(""), fmt.Errorf("missing saml2 definition reference")
			}

			ent := samlEntities[args[0].AsString()]
			if ent.err != nil {
				return cty.StringVal(""), ent.err
			}

			metadata := ent.descriptor
			var ssoUrl string
			for _, ssoService := range metadata.IDPSSODescriptor.SingleSignOnServices {
				if ssoService.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
					ssoUrl = ssoService.Location
					continue
				}
			}

			nameIDFormat := getNameIDFormat(metadata.IDPSSODescriptor.NameIDFormats)

			sp := &saml2.SAMLServiceProvider{
				AssertionConsumerServiceURL: ent.config.SpAcsUrl,
				IdentityProviderSSOURL:      ssoUrl,
				ServiceProviderIssuer:       ent.config.SpEntityId,
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
