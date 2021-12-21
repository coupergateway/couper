package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/oauth2/oidc"
)

func TestConfig_Synced(t *testing.T) {
	helper := test.New(t)

	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())

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
	defer origin.Close()

	be := transport.NewBackend(hcl.EmptyBody(), &transport.Config{}, nil, logger)
	o, err := oidc.NewConfig(&config.OIDC{ConfigurationURL: origin.URL + "/.well-known/openid-configuration"}, be)
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
}
