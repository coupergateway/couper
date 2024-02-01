package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
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

// IntrospectionResponse represents the response body to a token introspection request.
type IntrospectionResponse map[string]interface{}

func NewIntrospectionResponse(res *http.Response) (IntrospectionResponse, error) {
	var introspectionData IntrospectionResponse

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspection response status code %d", res.StatusCode)
	}

	if !eval.IsJSONMediaType(res.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("introspection response is not JSON")
	}

	err := json.NewDecoder(res.Body).Decode(&introspectionData)
	return introspectionData, err
}

// Active returns whether the token is active.
func (ir IntrospectionResponse) Active() bool {
	active, _ := ir["active"].(bool)
	return active
}

func (ir IntrospectionResponse) exp() int64 {
	exp, _ := ir["exp"].(int64)
	return exp
}

type lock struct {
	mu sync.Mutex
}

// Introspector represents a token introspector.
type Introspector struct {
	authenticator oauth2.ClientAuthenticator
	conf          *config.Introspection
	locks         sync.Map
	memStore      *cache.MemoryStore
	transport     http.RoundTripper
}

// NewIntrospector creates a new token introspector.
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

// Introspect retrieves introspection data for the given token using either cached or fresh information.
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
		cachedIntrospection, _ := i.memStore.Get(key).(IntrospectionResponse)
		if cachedIntrospection != nil {
			return cachedIntrospection, nil
		}
	}

	req, cancel, err := i.newIntrospectionRequest(ctx, token)
	defer cancel()
	if err != nil {
		return nil, err
	}

	introspectionData, err = i.requestIntrospection(req)
	if err != nil {
		return nil, err
	}

	if i.conf.TTLSeconds <= 0 {
		// do not cache
		return introspectionData, nil
	}

	if exp == 0 {
		if isdExp := introspectionData.exp(); isdExp > 0 {
			exp = isdExp
		}
	}

	ttl := i.getTtl(exp, nbf, introspectionData.Active())
	// cache introspection data
	i.memStore.Set(key, introspectionData, ttl)

	return introspectionData, nil
}

func (i *Introspector) newIntrospectionRequest(ctx context.Context, token string) (*http.Request, context.CancelFunc, error) {
	req, _ := http.NewRequest("POST", i.conf.Endpoint, nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	outCtx, cancel := context.WithCancel(context.WithValue(ctx, request.RoundTripName, "introspection"))
	outCtx = context.WithValue(outCtx, request.BufferOptions, buffer.Response)

	formParams := &url.Values{}
	formParams.Add("token", token)

	err := i.authenticator.Authenticate(formParams, req)
	if err != nil {
		return nil, cancel, err
	}

	eval.SetBody(req, []byte(formParams.Encode()))

	return req.WithContext(outCtx), cancel, nil
}

func (i *Introspector) requestIntrospection(req *http.Request) (IntrospectionResponse, error) {
	response, err := i.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("introspection response: %s", err)
	}
	defer response.Body.Close()

	introspectionData, err := NewIntrospectionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("introspection response: %s", err)
	}

	return introspectionData, nil
}

func (i *Introspector) getTtl(exp, nbf int64, active bool) int64 {
	ttl := i.conf.TTLSeconds

	if exp > 0 {
		now := time.Now().Unix()
		maxTTL := exp - now
		if !active && (nbf <= 0 || now > nbf) {
			// nbf is unknown (token has never been inactive before being active)
			// or nbf lies in the past (token has become active after having been inactive):
			// token will not become active again, so we can store the response until token expires anyway
			ttl = maxTTL
		} else if ttl > maxTTL {
			ttl = maxTTL
		}
	}
	return ttl
}
