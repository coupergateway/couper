package runtime

import (
	"fmt"
	"math"
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/oauth2"
)

func NewBackend(ctx *hcl.EvalContext, body hcl.Body, log *logrus.Entry,
	settings *config.Settings, store *cache.MemoryStore) (http.RoundTripper, hcl.Body, error) {
	const prefix = "backend_"
	name, err := getBackendName(ctx, body)

	if err != nil {
		return nil, nil, err
	}

	// Making use of the store here since a global variable leads to extra efforts for integration tests.
	// The store is newly created per run.
	backend := store.Get(prefix + name)
	if backend != nil {
		// TODO: check for merge bodies only?
		return transport.NewBackendContext(body, backend.(http.RoundTripper)), body, nil
	}

	ctxBody := body
	// prevent setting a context from the first anonymous backend, initialize with an empty body instead
	// if name == "default" {
	// 	ctxBody = hcl.EmptyBody()
	// }

	backend, modifiedBody, err := newBackend(ctx, ctxBody, log, settings.NoProxyFromEnv, settings.Certificate, store)
	if err != nil {
		return nil, nil, err
	}

	// to prevent weird debug sessions; max to set the internal memStore ttl limit.
	store.Set(prefix+name, backend, math.MaxInt64)

	return backend.(http.RoundTripper), modifiedBody, nil
}

func newBackend(evalCtx *hcl.EvalContext, backendCtx hcl.Body, log *logrus.Entry,
	ignoreProxyEnv bool, certificate []byte, memStore *cache.MemoryStore) (http.RoundTripper, hcl.Body, error) {
	beConf := &config.Backend{}
	if diags := gohcl.DecodeBody(backendCtx, evalCtx, beConf); diags.HasErrors() {
		return nil, nil, diags
	}

	if beConf.Name == "" {
		name, err := getBackendName(evalCtx, backendCtx)
		if err != nil {
			return nil, nil, err
		}
		beConf.Name = name
	}

	tc := &transport.Config{
		BackendName:            beConf.Name,
		Certificate:            certificate,
		DisableCertValidation:  beConf.DisableCertValidation,
		DisableConnectionReuse: beConf.DisableConnectionReuse,
		HTTP2:                  beConf.HTTP2,
		NoProxyFromEnv:         ignoreProxyEnv,
		MaxConnections:         beConf.MaxConnections,
	}

	openAPIopts, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	if err != nil {
		fmt.Printf("RUNTIME DONE '%#v' \n", err)
		return nil, nil, err
	}

	options := &transport.BackendOptions{
		OpenAPI: openAPIopts,
	}
	backend := transport.NewBackend(backendCtx, tc, options, log)

	oauthContent, _, _ := backendCtx.PartialContent(config.OAuthBlockSchema)
	if oauthContent == nil {
		return backend, backendCtx, nil
	}

	if blocks := oauthContent.Blocks.OfType("oauth2"); len(blocks) > 0 {
		return newAuthBackend(evalCtx, beConf, blocks, log, ignoreProxyEnv, certificate, memStore, backend)
	}

	return backend, backendCtx, nil
}

func newAuthBackend(evalCtx *hcl.EvalContext, beConf *config.Backend, blocks hcl.Blocks, log *logrus.Entry,
	ignoreProxyEnv bool, certificate []byte, memStore *cache.MemoryStore, backend http.RoundTripper) (http.RoundTripper, hcl.Body, error) {

	beConf.OAuth2 = &config.OAuth2ReqAuth{}

	if diags := gohcl.DecodeBody(blocks[0].Body, evalCtx, beConf.OAuth2); diags.HasErrors() {
		return nil, nil, diags
	}

	innerContent, _, diags := beConf.OAuth2.Remain.PartialContent(beConf.OAuth2.Schema(true))
	if diags.HasErrors() {
		return nil, nil, diags
	}

	innerBackend := innerContent.Blocks.OfType("backend")[0] // backend block is set by configload
	authBackend, body, authErr := newBackend(evalCtx, innerBackend.Body, log, ignoreProxyEnv, certificate, memStore)
	if authErr != nil {
		return nil, nil, authErr
	}

	// Set default value
	if beConf.OAuth2.Retries == nil {
		var one uint8 = 1
		beConf.OAuth2.Retries = &one
	}

	oauth2Client, err := oauth2.NewClientCredentialsClient(beConf.OAuth2, authBackend)
	if err != nil {
		return nil, body, err
	}

	rt, err := transport.NewOAuth2ReqAuth(beConf.OAuth2, memStore, oauth2Client, backend)
	return rt, body, err
}

func getBackendName(evalCtx *hcl.EvalContext, backendCtx hcl.Body) (string, error) {
	content, _, _ := backendCtx.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{
		{Name: "name"}},
	})

	if content != nil && len(content.Attributes) > 0 {
		if n, exist := content.Attributes["name"]; exist {
			v, err := eval.Value(evalCtx, n.Expr)
			if err != nil {
				return "", err
			}

			return v.AsString(), nil
		}
	}

	return "", nil
}
