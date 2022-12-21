package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
)

type Introspector struct {
	conf      *config.Introspection
	memStore  *cache.MemoryStore
	transport http.RoundTripper
}

func NewIntrospector(conf *config.Introspection, transport http.RoundTripper, memStore *cache.MemoryStore) *Introspector {
	return &Introspector{
		conf:      conf,
		memStore:  memStore,
		transport: transport,
	}
}

type IntrospectionResponse map[string]interface{}

func (ir IntrospectionResponse) Active() bool {
	active, _ := ir["active"].(bool)
	return active
}

func (ir IntrospectionResponse) Exp() int64 {
	exp, _ := ir["exp"].(int64)
	return exp
}

func (i *Introspector) Introspect(ctx context.Context, token string, exp, nbf int64) (IntrospectionResponse, error) {
	key := "ir:" + token

	var introspectionData IntrospectionResponse

	ttl := i.conf.TTLSeconds

	if ttl > 0 {
		cachedIntrospectionBytes, _ := i.memStore.Get(key).([]byte)
		if cachedIntrospectionBytes != nil {
			// cached introspection response is always JSON
			_ = json.Unmarshal(cachedIntrospectionBytes, &introspectionData)

			return introspectionData, nil
		}
	}

	req, _ := http.NewRequest("POST", i.conf.Endpoint, nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	outCtx, cancel := context.WithCancel(context.WithValue(ctx, request.RoundTripName, "introspection"))
	defer cancel()
	outCtx = context.WithValue(outCtx, request.BufferOptions, buffer.Response)

	formParams := &url.Values{}
	formParams.Add("token", token)
	eval.SetBody(req, []byte(formParams.Encode()))

	req = req.WithContext(outCtx)

	response, err := i.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("introspection response: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspection response status code %d", response.StatusCode)
	}

	resBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("introspection response cannot be read: %s", err)
	}

	err = json.Unmarshal(resBytes, &introspectionData)
	if err != nil {
		return nil, fmt.Errorf("introspection response is not JSON: %s", err)
	}

	if ttl <= 0 {
		return introspectionData, nil
	}

	if exp == 0 {
		if isdExp := introspectionData.Exp(); isdExp > 0 {
			exp = isdExp
		}
	}

	if exp > 0 {
		now := time.Now().Unix()
		maxTTL := exp - now
		if !introspectionData.Active() && (nbf <= 0 || now > nbf) {
			// nbf is unknown (token has never been inactive before being active)
			// or nbf lies in the past (token has become active after having been inactive):
			// token will not become active again, so we can store the response until token expires anyway
			ttl = maxTTL
		} else if ttl > maxTTL {
			ttl = maxTTL
		}
	}
	i.memStore.Set(key, resBytes, ttl)

	return introspectionData, nil
}
