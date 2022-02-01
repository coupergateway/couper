package lib_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/oauth2/oidc"
)

func TestNewOAuthAuthorizationUrlFunction(t *testing.T) {
	helper := test.New(t)

	expFn := func(exp string) hcl.Expression {
		e, diags := hclsyntax.ParseExpression([]byte(exp), "", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatal(diags)
		}
		return e
	}

	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var conf interface{}
		if req.URL.Path == "/.well-known/openid-configuration" {
			conf = &oidc.OpenidConfiguration{
				AuthorizationEndpoint:         origin.URL + "/auth",
				CodeChallengeMethodsSupported: []string{config.CcmS256},
				Issuer:                        "thatsme",
				JwksUri:                       origin.URL + "/jwks",
				TokenEndpoint:                 origin.URL + "/token",
				UserinfoEndpoint:              origin.URL + "/userinfo",
			}
		} else if req.URL.Path == "/jwks" {
			conf = jwk.JWKSData{}
		}
		b, err := json.Marshal(conf)
		helper.Must(err)
		_, err = rw.Write(b)
		helper.Must(err)
	}))

	backendConfig, _ := configload.NewNamedBody("origin", test.NewRemainContext("origin", origin.URL))

	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())

	tests := []struct {
		name         string
		oauth2Config *config.OIDC
		want         string
	}{
		{
			name: "redirect_uri with client request",
			oauth2Config: &config.OIDC{
				Name:             "auth-ref",
				ConfigurationURL: origin.URL + "/.well-known/openid-configuration",
				Remain: hclbody.New(&hcl.BodyContent{
					Attributes: map[string]*hcl.Attribute{
						lib.RedirectURI: {
							Name: lib.RedirectURI,
							Expr: expFn("request.headers.x-want"),
						},
					}})},
			want: "https://couper.io/cb",
		},
		{
			name: "redirect_uri with backend_requests", // works since client and backend request are the same at response obj
			oauth2Config: &config.OIDC{
				Name:             "auth-ref",
				ConfigurationURL: origin.URL + "/.well-known/openid-configuration",
				Remain: hclbody.New(&hcl.BodyContent{
					Attributes: map[string]*hcl.Attribute{
						lib.RedirectURI: {
							Name: lib.RedirectURI,
							Expr: expFn("backend_requests.default.headers.x-want"),
						},
					}})},
			want: "https://couper.io/cb",
		},
		{
			name: "redirect_uri with backend_responses",
			oauth2Config: &config.OIDC{
				Name:             "auth-ref",
				ConfigurationURL: origin.URL + "/.well-known/openid-configuration",
				Remain: hclbody.New(&hcl.BodyContent{
					Attributes: map[string]*hcl.Attribute{
						lib.RedirectURI: {
							Name: lib.RedirectURI,
							Expr: expFn("backend_responses.default.headers.x-want"),
						},
					}})},
			want: "https://couper.io/cb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "https://couper.io/", nil)
			helper.Must(err)
			*req = *req.Clone(context.Background())
			req.Header.Set("x-want", tt.want)

			res := &http.Response{
				Header:     make(http.Header),
				Request:    req,
				StatusCode: http.StatusNoContent,
			}
			res.Header.Set("x-want", tt.want)

			tc := &transport.Config{}

			backend := transport.NewBackend(backendConfig,
				tc.With("http", "couper.io", "couper.io", ""),
				&transport.BackendOptions{}, logger)

			// TODO: call prepare iface instead
			backends := map[string]http.RoundTripper{
				"configuration_backend": backend,
			}

			conf, err := oidc.NewConfig(tt.oauth2Config, backends)
			helper.Must(err)

			ctx := eval.NewContext(nil, &config.Defaults{}).
				WithOidcConfig(oidc.Configs{conf.Name: conf}).
				WithClientRequest(req).
				WithBeresp(res, false)

			hclCtx := ctx.HCLContext()
			val, err := hclCtx.Functions[lib.FnOAuthAuthorizationUrl].Call([]cty.Value{cty.StringVal("auth-ref")})
			helper.Must(err)

			authUrl := seetie.ValueToString(val)
			authUrlObj, err := url.Parse(authUrl)
			helper.Must(err)

			if authUrlObj.Query().Get(lib.RedirectURI) != tt.want {
				subT.Errorf("Want: %v; got: %v", tt.want, val)
			}
		})
	}
}
