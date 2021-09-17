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

	"github.com/avenga/couper/cache"
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

	expFn := func(exp string) hcl.Expression {
		e, diags := hclsyntax.ParseExpression([]byte(exp), "", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatal(diags)
		}
		return e
	}

	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		conf := &oidc.OpenidConfiguration{
			AuthorizationEndpoint:         origin.URL + "/auth",
			Issuer:                        "me",
			TokenEndpoint:                 origin.URL + "/token",
			UserinfoEndpoint:              origin.URL + "/userinfo",
			CodeChallengeMethodsSupported: []string{"S256"},
		}
		b, _ := json.Marshal(conf)
		_, _ = rw.Write(b)
	}))

	backend := configload.NewBackend("origin", test.NewRemainContext("origin", origin.URL))

	memQuitCh := make(chan struct{})
	defer close(memQuitCh)
	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())
	memStore := cache.New(logger, memQuitCh)

	tests := []struct {
		name         string
		oauth2Config *config.OIDC
		want         string
	}{{name: "redirect_uri with client request",
		oauth2Config: &config.OIDC{
			Name: "auth-ref",
			Remain: hclbody.New(&hcl.BodyContent{
				Attributes: map[string]*hcl.Attribute{
					lib.RedirectURI: {
						Name: lib.RedirectURI,
						Expr: expFn("request.headers.x-want"),
					},
				}})},
		want: "https://couper.io/cb",
	},
		{name: "redirect_uri with backend_requests", // works since client and backend request are the same at response obj
			oauth2Config: &config.OIDC{
				Name: "auth-ref",
				Remain: hclbody.New(&hcl.BodyContent{
					Attributes: map[string]*hcl.Attribute{
						lib.RedirectURI: {
							Name: lib.RedirectURI,
							Expr: expFn("backend_requests.default.headers.x-want"),
						},
					}})},
			want: "https://couper.io/cb",
		},
		{name: "redirect_uri with backend_responses",
			oauth2Config: &config.OIDC{
				Name: "auth-ref",
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
		t.Run(tt.name, func(st *testing.T) {
			helper := test.New(st)

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
			conf, err := oidc.NewConfig(tt.oauth2Config, transport.NewBackend(backend.Config, eval.NewContext(nil, nil),
				tc.With("http", "couper.io", "couper.io", ""),
				&transport.BackendOptions{}, logger), memStore)
			helper.Must(err)
			conf.ConfigurationURL = origin.URL

			ctx := eval.NewContext(nil, &config.Defaults{}).
				WithOidcConfig(oidc.Configs{conf.Name: conf}).
				WithClientRequest(req).
				WithBeresps(res)

			hclCtx := ctx.HCLContext()
			val, err := hclCtx.Functions[lib.FnOAuthAuthorizationUrl].Call([]cty.Value{cty.StringVal("auth-ref")})
			helper.Must(err)

			authUrl := seetie.ValueToString(val)
			authUrlObj, err := url.Parse(authUrl)
			helper.Must(err)

			if authUrlObj.Query().Get(lib.RedirectURI) != tt.want {
				t.Errorf("Want: %v; got: %v", tt.want, val)
			}
		})
	}
}
