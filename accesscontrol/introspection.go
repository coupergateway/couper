package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/oauth2"
)

type lock struct {
	mu sync.Mutex
}

type Introspector struct {
	authenticator *oauth2.ClientAuthenticator
	conf          *config.Introspection
	locks         sync.Map
	memStore      *cache.MemoryStore
	transport     http.RoundTripper
}

func NewIntrospector(evalCtx *hcl.EvalContext, conf *config.Introspection, transport http.RoundTripper, memStore *cache.MemoryStore) (*Introspector, error) {
	authenticator, err := oauth2.NewClientAuthenticator(evalCtx, conf.EndpointAuthMethod, "endpoint_auth_method", conf.ClientID, conf.ClientSecret, "", conf.JWTSigningProfile)
	if err != nil {
		return nil, err
	}
	return &Introspector{
		authenticator: authenticator,
		conf:          conf,
		memStore:      memStore,
		transport:     transport,
	}, nil
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
	var (
		introspectionData IntrospectionResponse
		key               string
	)

	if i.conf.TTLSeconds > 0 {
		// lock per token
		entry, _ := i.locks.LoadOrStore(token, &lock{})
		l := entry.(*lock)
		l.mu.Lock()
		defer func() {
			i.locks.Delete(token)
			l.mu.Unlock()
		}()

		key = "ir:" + token
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

	err := i.authenticator.Authenticate(formParams, req)
	if err != nil {
		return nil, err
	}

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

	if i.conf.TTLSeconds <= 0 {
		return introspectionData, nil
	}

	if exp == 0 {
		if isdExp := introspectionData.Exp(); isdExp > 0 {
			exp = isdExp
		}
	}

	ttl := i.conf.TTLSeconds

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
