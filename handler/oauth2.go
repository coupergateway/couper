package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/logging"
)

type oAuth2 struct {
	config    *config.OAuth2
	proxy     *Proxy
	transport *transportConfig
}

func newOAuth2(proxy *Proxy, config *config.OAuth2, transport *transportConfig) *oAuth2 {
	return &oAuth2{
		config:    config,
		proxy:     proxy,
		transport: transport,
	}
}

func (oa *oAuth2) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if oa.config == nil {
		oa.gotoProxy(rw, req, nil)
		return
	}

	user, pass, err := oa.getCredentials(req)
	if err != nil {
		oa.gotoProxy(rw, req, err)
		return
	}

	key := oa.config.TokenEndpoint + "|" + user + "|" + pass

	memStore := req.Context().Value(request.MemStore).(*cache.MemoryStore)
	if data := memStore.Get(key); data != "" {
		token, err := oa.getAccessToken(data, key, nil)
		if err != nil {
			oa.gotoProxy(rw, req, err)
			return
		}

		req.Header.Set("Authorization", "Bearer "+token)

		oa.gotoProxy(rw, req, nil)
		return
	}

	params := "grant_type=client_credentials" + "&" +
		"token_endpoint_auth_method=client_secret_basic"

	body := ioutil.NopCloser(strings.NewReader(params))
	outreq, err := http.NewRequest("POST", oa.config.TokenEndpoint, body)
	if err != nil {
		oa.gotoProxy(rw, req, err)
		return
	}

	auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	outreq.Header.Set("Authorization", "Basic "+auth)
	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	conf := oa.transport
	conf.hash = key

	res, err := getTransport(conf).RoundTrip(outreq)
	if err != nil {
		oa.gotoProxy(rw, req, err)
		return
	}

	if res.StatusCode != http.StatusOK {
		oa.gotoProxy(rw, req, nil)
		return
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		oa.gotoProxy(rw, req, err)
		return
	}

	token, err := oa.getAccessToken(string(b), key, memStore)
	if err != nil {
		oa.gotoProxy(rw, req, err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)

	oa.gotoProxy(rw, req, nil)
}

func (oa *oAuth2) getCredentials(req *http.Request) (string, string, error) {
	content, _, diags := oa.config.Remain.PartialContent(
		oa.config.Schema(true),
	)
	if diags.HasErrors() {
		return "", "", diags
	}

	evalCtx := eval.NewHTTPContext(
		oa.proxy.evalContext, oa.proxy.bufferOption, req, nil, nil,
	)

	ua, userOK := content.Attributes["client_id"]
	uv, _ := ua.Expr.Value(evalCtx)
	user := seetie.ValueToString(uv)

	pa, passOK := content.Attributes["client_secret"]
	pv, _ := pa.Expr.Value(evalCtx)
	pass := seetie.ValueToString(pv)

	if !userOK || !passOK {
		return "", "", fmt.Errorf("Missing OAuth2 'client_id' or 'client_secret' value")
	}

	return user, pass, nil
}

func (oa *oAuth2) gotoProxy(rw http.ResponseWriter, req *http.Request, err error) {
	startTime := time.Now()

	if err != nil {
		oa.proxy.log.Error(err)
	}

	oa.proxy.upstreamLog.ServeHTTP(rw, req, logging.RoundtripHandlerFunc(oa.proxy.roundtrip), startTime)
}

func (oa *oAuth2) getAccessToken(jsonString, key string, memStore *cache.MemoryStore) (string, error) {
	var jData map[string]interface{}

	err := json.Unmarshal([]byte(jsonString), &jData)
	if err != nil {
		return "", err
	}

	var token string
	if t, ok := jData["access_token"].(string); ok {
		token = t
	} else {
		return "", fmt.Errorf("Missing OAuth2 'access_token'")
	}

	if memStore != nil {
		var ttl int64
		if t, ok := jData["expires_in"].(float64); ok {
			ttl = (int64)(t * 0.9)
		}

		memStore.Set(key, jsonString, time.Now().Unix()+ttl)
	}

	return token, nil
}
