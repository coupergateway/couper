package lib_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/oauth2/oidc"
)

func TestNewOAuthAuthorizationURLFunction(t *testing.T) {
	helper := test.New(t)

	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var conf interface{}
		if req.URL.Path == "/.well-known/openid-configuration" {
			conf = &oidc.OpenidConfiguration{
				AuthorizationEndpoint:         origin.URL + "/auth",
				CodeChallengeMethodsSupported: []string{config.CcmS256},
				Issuer:                        "thatsme",
				JwksURI:                       origin.URL + "/jwks",
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
	u := origin.URL + "/.well-known/openid-configuration"
	backendConfig := ` server {}
definitions {
  oidc "auth-ref" {
 	client_id = "test-id"
	client_secret = "test-s3cr3t"
    configuration_url = "` + u + `"
	redirect_uri = "${request.headers.x-want},${backend_requests.default.headers.x-want},${backend_responses.default.headers.x-want}"
	verifier_value = "asdf"
  }
}
`

	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())

	couperConf, err := configload.LoadBytes([]byte(backendConfig), "test.hcl")
	helper.Must(err)

	quitCh := make(chan struct{}, 1)
	defer close(quitCh)
	memStore := cache.New(logger, quitCh)

	ctx, cancel := context.WithCancel(couperConf.Context)
	couperConf.Context = ctx
	defer cancel()

	_, err = runtime.NewServerConfiguration(couperConf, logger, memStore)
	helper.Must(err)

	// redirect_uri = "${request.headers.x-want},${backend_requests.default.headers.x-want},${backend_responses.default.headers.x-want}"
	want := "https://couper.io/cb,https://couper.io/cb,https://couper.io/cb"

	req, rerr := http.NewRequest(http.MethodGet, "https://couper.io/", nil)
	helper.Must(rerr)
	req = req.Clone(context.Background())
	req.Header.Set("x-want", "https://couper.io/cb")

	res := &http.Response{
		Header:     http.Header{"x-want": []string{"https://couper.io/cb"}},
		Request:    req,
		StatusCode: http.StatusNoContent,
	}

	hclCtx := couperConf.Context.(*eval.Context).
		WithClientRequest(req).
		WithBeresp(res, false).
		HCLContext()

	val, furr := hclCtx.Functions[lib.FnOAuthAuthorizationURL].Call([]cty.Value{cty.StringVal("auth-ref")})
	helper.Must(furr)

	authURL := seetie.ValueToString(val)
	authURLObj, perr := url.Parse(authURL)
	helper.Must(perr)

	if value := authURLObj.Query().Get(lib.RedirectURI); value != want {
		t.Errorf("Want: %v; got: %v", want, value)
	}
}
