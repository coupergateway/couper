package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/accesscontrol/jwk"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/handler/transport"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/oauth2/oidc"
)

func TestConfig_Synced(t *testing.T) {
	helper := test.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, _ := test.NewLogger()
	logger := log.WithContext(ctx)

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
	defer origin.Close()

	oconf := &config.OIDC{
		ConfigurationURL: origin.URL + "/.well-known/openid-configuration",
		ConfigurationTTL: "100ms",
		Remain: &hclsyntax.Body{Attributes: hclsyntax.Attributes{
			"redirect_uri":   {Name: "redirect_uri", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("")}},
			"verifier_value": {Name: "verifier_value", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("")}},
		}},
	}
	// configload internals here... TODO: integration test?
	err := oconf.Prepare(func(attr string, val string, body config.Body) (*hclsyntax.Body, error) {
		return configload.PrepareBackend(nil, attr, val, body)
	})
	helper.Must(err)

	backends := make(map[string]http.RoundTripper)
	for k, b := range oconf.Backends {
		backends[k] = transport.NewBackend(b, &transport.Config{}, nil, logger)
	}

	o, err := oidc.NewConfig(ctx, oconf, backends)
	helper.Must(err)

	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer wg.Done()

			_, e := o.GetIssuer()
			helper.Must(e)
		}(i)
	}
	wg.Wait()

	// wait for possible goroutine leaks from syncedResource due to low ttl
	time.Sleep(time.Second / 2)

	if n := test.NumGoroutines("resource.(*SyncedResource).sync"); n != 2 {
		t.Errorf("Expected two running routines, got: %d", n)
	}
}
