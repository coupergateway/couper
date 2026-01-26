package lib_test

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/accesscontrol/saml"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/test"
)

func Test_SamlSsoURL(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		samlLabel string
		wantPfx   string
	}{
		{
			"metadata found",
			`
			server "test" {
			}
			definitions {
				saml "MySAML" {
					idp_metadata_file = "testdata/idp-metadata.xml"
					sp_entity_id = "the-sp"
					sp_acs_url = "https://sp.example.com/saml/acs"
					array_attributes = ["memberOf"]
				}
			}
			`,
			"MySAML",
			"https://idp.example.org/saml/SSOService",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := test.New(subT)
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				h.Must(err)
			}

			// Create SAML providers from loaded config
			providers := make(map[string]lib.SAMLConfigWithProvider)
			for _, samlConf := range cf.Definitions.SAML {
				provider, provErr := saml.NewStaticMetadata(samlConf.MetadataBytes)
				h.Must(provErr)
				providers[samlConf.Name] = lib.SAMLConfigWithProvider{
					Config:   samlConf,
					Provider: provider,
				}
			}

			evalContext := cf.Context.Value(request.ContextType).(*eval.Context)
			evalContext = evalContext.WithSAMLProviders(providers)

			req, err := http.NewRequest(http.MethodGet, "https://www.example.com/foo", nil)
			h.Must(err)
			evalContext = evalContext.WithClientRequest(req)

			ssoURL, err := evalContext.HCLContext().Functions[lib.FnSamlSsoURL].Call([]cty.Value{cty.StringVal(tt.samlLabel)})
			h.Must(err)

			if !strings.HasPrefix(ssoURL.AsString(), tt.wantPfx) {
				subT.Errorf("Expected to start with %q, got: %#v", tt.wantPfx, ssoURL.AsString())
			}

			u, err := url.Parse(ssoURL.AsString())
			h.Must(err)

			q := u.Query()
			samlRequest := q.Get("SAMLRequest")
			if samlRequest == "" {
				subT.Fatal("Expected SAMLRequest query param")
			}

			b64Decoded, err := base64.StdEncoding.DecodeString(samlRequest)
			h.Must(err)

			fr := flate.NewReader(bytes.NewReader(b64Decoded))
			deflated, err := io.ReadAll(fr)
			h.Must(err)

			var x interface{}
			err = xml.Unmarshal(deflated, &x)
			h.Must(err)
		})
	}
}

func TestSamlConfigError(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		label   string
		wantErr string
	}{
		{
			"missing referenced saml IdP metadata",
			`
			server {}
			definitions {
			  saml "MySAML" {
			    idp_metadata_file = "/not/there"
			    sp_entity_id = "the-sp"
			    sp_acs_url = "https://sp.example.com/saml/acs"
			  }
			}
			`,
			"MyLabel",
			"configuration error: MySAML: saml2 idp_metadata_file: read error: open /not/there: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := configload.LoadBytes([]byte(tt.config), "test.hcl")
			if err == nil {
				subT.Error("expected an error, got nothing")
				return
			}
			gErr := err.(errors.GoError)
			if gErr.LogError() != tt.wantErr {
				subT.Errorf("\nWant:\t%q\nGot:\t%q", tt.wantErr, gErr.LogError())
			}
		})
	}
}

func TestSamlSsoURLError(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		label   string
		wantErr string
	}{
		{
			"missing saml definitions",
			`
			server {}
			definitions {
			}
			`,
			"MyLabel",
			`missing saml block with referenced label "MyLabel"`,
		},
		{
			"missing referenced saml",
			`
			server {}
			definitions {
			  saml "MySAML" {
			    idp_metadata_file = "testdata/idp-metadata.xml"
			    sp_entity_id = "the-sp"
			    sp_acs_url = "https://sp.example.com/saml/acs"
			  }
			}
			`,
			"MyLabel",
			`missing saml block with referenced label "MyLabel"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := test.New(subT)
			couperConf, err := configload.LoadBytes([]byte(tt.config), "test.hcl")
			h.Must(err)

			ctx, cancel := context.WithCancel(couperConf.Context)
			couperConf.Context = ctx
			defer cancel()

			// Create SAML providers from loaded config
			providers := make(map[string]lib.SAMLConfigWithProvider)
			for _, samlConf := range couperConf.Definitions.SAML {
				provider, provErr := saml.NewStaticMetadata(samlConf.MetadataBytes)
				h.Must(provErr)
				providers[samlConf.Name] = lib.SAMLConfigWithProvider{
					Config:   samlConf,
					Provider: provider,
				}
			}

			evalContext := couperConf.Context.Value(request.ContextType).(*eval.Context)
			evalContext = evalContext.WithSAMLProviders(providers)

			req, err := http.NewRequest(http.MethodGet, "https://www.example.com/foo", nil)
			h.Must(err)
			evalContext = evalContext.WithClientRequest(req)

			_, err = evalContext.HCLContext().Functions[lib.FnSamlSsoURL].Call([]cty.Value{cty.StringVal(tt.label)})
			if err == nil {
				subT.Error("expected an error, got nothing")
				return
			}
			if err.Error() != tt.wantErr {
				subT.Errorf("\nWant:\t%q\nGot:\t%q", tt.wantErr, err.Error())
			}
		})
	}
}
