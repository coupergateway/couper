package lib_test

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/test"
)

func Test_SamlSsoUrl(t *testing.T) {
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
		{
			"label mismatch",
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
			"NotThere",
			true,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			h := test.New(st)
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				if tt.wantErr {
					return
				}
				h.Must(err)
			}

			hclContext := cf.Context.Value(eval.ContextType).(*eval.Context).HCLContext()

			ssoUrl, err := hclContext.Functions[lib.FnSamlSsoUrl].Call([]cty.Value{cty.StringVal(tt.samlLabel)})
			if err == nil && tt.wantErr {
				st.Fatal("Error expected")
			}
			if err != nil {
				if !tt.wantErr {
					h.Must(err)
				} else {
					return
				}
			}

			if !strings.HasPrefix(ssoUrl.AsString(), tt.wantPfx) {
				st.Errorf("Expected to start with %q, got: %#v", tt.wantPfx, ssoUrl.AsString())
			}

			u, err := url.Parse(ssoUrl.AsString())
			h.Must(err)

			q := u.Query()
			samlRequest := q.Get("SAMLRequest")
			if samlRequest == "" {
				st.Fatal("Expected SAMLRequest query param")
			}

			b64Decoded, err := base64.StdEncoding.DecodeString(samlRequest)
			h.Must(err)

			fr := flate.NewReader(bytes.NewReader(b64Decoded))
			deflated, err := ioutil.ReadAll(fr)
			h.Must(err)

			var x interface{}
			err = xml.Unmarshal(deflated, &x)
			h.Must(err)
		})
	}

}
