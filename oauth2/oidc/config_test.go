package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/avenga/couper/eval"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/oauth2/oidc"
)

func TestConfig_getFreshIfExpiredSynced(t *testing.T) {
	helper := test.New(t)

	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())

	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		conf := &oidc.OpenidConfiguration{
			AuthorizationEndpoint:         origin.URL + "/auth",
			Issuer:                        "thatsme",
			TokenEndpoint:                 origin.URL + "/token",
			UserinfoEndpoint:              origin.URL + "/userinfo",
			CodeChallengeMethodsSupported: []string{config.CcmS256},
		}

		b, err := json.Marshal(conf)
		helper.Must(err)
		_, err = rw.Write(b)
		helper.Must(err)
	}))
	defer origin.Close()

	memQuitCh := make(chan struct{})
	defer close(memQuitCh)
	memStore := cache.New(logger, memQuitCh)

	be := transport.NewBackend(configload.EmptyBody(), eval.NewContext(nil, nil), &transport.Config{}, nil, logger)
	o, err := oidc.NewConfig(&config.OIDC{ConfigurationURL: origin.URL}, be, memStore)
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
