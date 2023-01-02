package lib_test

import (
	"bytes"
	"context"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/test"
)

func Test_SamlSsoURL(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		samlLabel string
		wantErr   bool
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
			false,
			"https://idp.example.org/saml/SSOService",
		},
		{
			"metadata not found",
			`
			server "test" {
			}
			definitions {
				saml "MySAML" {
					idp_metadata_file = "not-there"
					sp_entity_id = "the-sp"
					sp_acs_url = "https://sp.example.com/saml/acs"
					array_attributes = ["memberOf"]
				}
			}
			`,
			"MySAML",
			true,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := test.New(subT)
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				if tt.wantErr {
					return
				}
				h.Must(err)
			}

			evalContext := cf.Context.Value(request.ContextType).(*eval.Context)
			req, err := http.NewRequest(http.MethodGet, "https://www.example.com/foo", nil)
			h.Must(err)
			evalContext = evalContext.WithClientRequest(req)

			ssoURL, err := evalContext.HCLContext().Functions[lib.FnSamlSsoURL].Call([]cty.Value{cty.StringVal(tt.samlLabel)})
			if err == nil && tt.wantErr {
				subT.Fatal("Error expected")
			}
			if err != nil {
				if !tt.wantErr {
					h.Must(err)
				} else {
					return
				}
			}

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

func TestSamlSsoURLError(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		label    string
		wantErr  string
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

			evalContext := couperConf.Context.Value(request.ContextType).(*eval.Context)
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
