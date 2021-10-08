package lib

import (
	"testing"

	"github.com/russellhaering/gosaml2/types"
)

func Test_getNameIDFormat(t *testing.T) {
	tests := []struct {
		name             string
		supportedFormats []types.NameIDFormat
		wantFormat       string
	}{
		{
			"only unspecified",
			[]types.NameIDFormat{
				{Value: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"},
			},
			"urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
		},
		{
			"unspecified 1st",
			[]types.NameIDFormat{
				{Value: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"},
				{Value: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"},
				{Value: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"},
			},
			"urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
		},
		{
			"unspecified 2nd",
			[]types.NameIDFormat{
				{Value: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"},
				{Value: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"},
				{Value: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"},
			},
			"urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
		},
		{
			"no unspecified",
			[]types.NameIDFormat{
				{Value: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"},
				{Value: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"},
				{Value: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"},
			},
			"urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			format := getNameIDFormat(tt.supportedFormats)
			if format != tt.wantFormat {
				subT.Errorf("Expected format %q, got: %#v", tt.wantFormat, format)
			}
		})
	}
}
