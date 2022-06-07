package runtime

import (
	"fmt"
	"math"
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/backend"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
)

func NewBackend(ctx *hcl.EvalContext, body hcl.Body, log *logrus.Entry,
	conf *config.Couper, store *cache.MemoryStore) (http.RoundTripper, error) {
	const prefix = "backend_"
	name, err := getBackendName(ctx, body)

	if err != nil {
		return nil, err
	}

	// Making use of the store here since a global variable leads to extra efforts for integration tests.
	// The store is newly created per run.
	b := store.Get(prefix + name)
	if b != nil {
		return backend.NewContext(body, b.(http.RoundTripper)), nil
	}

	b, err = newBackend(ctx, body, log, conf, store)
	if err != nil {
		return nil, err
	}

	// to prevent weird debug sessions; max to set the internal memStore ttl limit.
	store.Set(prefix+name, b, math.MaxInt64)

	return b.(http.RoundTripper), nil
}

func newBackend(evalCtx *hcl.EvalContext, backendCtx hcl.Body, log *logrus.Entry,
	conf *config.Couper, memStore *cache.MemoryStore) (http.RoundTripper, error) {
	beConf := &config.Backend{}
	if diags := gohcl.DecodeBody(backendCtx, evalCtx, beConf); diags.HasErrors() {
		return nil, diags
	}

	if beConf.Name == "" {
		name, err := getBackendName(evalCtx, backendCtx)
		if err != nil {
			return nil, err
		}
		beConf.Name = name
	}

	tc := &transport.Config{
		BackendName:            beConf.Name,
		Certificate:            conf.Settings.Certificate,
		DisableCertValidation:  beConf.DisableCertValidation,
		DisableConnectionReuse: beConf.DisableConnectionReuse,
		HTTP2:                  beConf.HTTP2,
		NoProxyFromEnv:         conf.Settings.NoProxyFromEnv,
		MaxConnections:         beConf.MaxConnections,
	}

	options := &transport.BackendOptions{}
	var err error
	options.OpenAPI, err = validation.NewOpenAPIOptions(beConf.OpenAPI)
	if err != nil {
		return nil, err
	}

	if beConf.Health != nil {
		origin, diags := eval.ValueFromBodyAttribute(evalCtx, backendCtx, "origin")
		if diags != nil {
			return nil, diags
		}

		if origin == cty.NilVal {
			return nil, fmt.Errorf("missing origin for backend %q", beConf.Name)
		}

		options.HealthCheck, err = config.NewHealthCheck(origin.AsString(), beConf.Health, conf)
		if err != nil {
			return nil, err
		}
	}

	oauthContent, _, _ := backendCtx.PartialContent(config.OAuthBlockSchema)
	if oauthContent != nil {
		if blocks := oauthContent.Blocks.OfType("oauth2"); len(blocks) > 0 {
			options.RequestAuthz, err = newOAuth2RequestAuthorizer(evalCtx, beConf, blocks, log, conf, memStore)
			if err != nil {
				return nil, err
			}
		}
	}
	tokenRequestContent, _, _ := backendCtx.PartialContent(config.TokenRequestBlockSchema)
	if tokenRequestContent != nil {
		if blocks := tokenRequestContent.Blocks.OfType("token_request"); len(blocks) > 0 {
			options.RequestAuthz, err = newTokenRequestAuthorizer(evalCtx, beConf, blocks, log, conf, memStore)
			if err != nil {
				return nil, err
			}
		}
	}

	b := transport.NewBackend(backendCtx, tc, options, log)
	return b, nil
}

func newOAuth2RequestAuthorizer(evalCtx *hcl.EvalContext, beConf *config.Backend, blocks hcl.Blocks, log *logrus.Entry,
	conf *config.Couper, memStore *cache.MemoryStore) (transport.RequestAuthorizer, error) {

	beConf.OAuth2 = &config.OAuth2ReqAuth{}

	if diags := gohcl.DecodeBody(blocks[0].Body, evalCtx, beConf.OAuth2); diags.HasErrors() {
		return nil, diags
	}

	innerContent, _, diags := beConf.OAuth2.Remain.PartialContent(beConf.OAuth2.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	backendBlocks := innerContent.Blocks.OfType("backend")
	if len(backendBlocks) == 0 {
		r := beConf.OAuth2.Remain.MissingItemRange()
		diag := &hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &r,
			Summary: "missing backend initialization",
		}}
		return nil, errors.Configuration.Label("unexpected").With(diag)
	}
	innerBackend := backendBlocks[0] // backend block is set by configload
	authBackend, authErr := NewBackend(evalCtx, innerBackend.Body, log, conf, memStore)
	if authErr != nil {
		return nil, authErr
	}

	// Set default value
	if beConf.OAuth2.Retries == nil {
		var one uint8 = 1
		beConf.OAuth2.Retries = &one
	}

	tr, err := transport.NewOAuth2ReqAuth(beConf.OAuth2, memStore, authBackend)
	if err != nil {
		return nil, errors.Backend.Label(beConf.Name).With(err)
	}

	return tr, nil
}

func newTokenRequestAuthorizer(evalCtx *hcl.EvalContext, beConf *config.Backend, blocks hcl.Blocks, log *logrus.Entry,
	conf *config.Couper, memStore *cache.MemoryStore) (transport.RequestAuthorizer, error) {

	trConfig := &config.TokenRequest{}

	if diags := gohcl.DecodeBody(blocks[0].Body, evalCtx, trConfig); diags.HasErrors() {
		return nil, diags
	}

	innerContent, _, diags := trConfig.Remain.PartialContent(trConfig.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	innerBackend := innerContent.Blocks.OfType("backend")[0] // backend block is set by configload
	authBackend, authErr := NewBackend(evalCtx, innerBackend.Body, log, conf, memStore)
	if authErr != nil {
		return nil, authErr
	}

	tr, err := transport.NewTokenRequest(trConfig, memStore, authBackend, beConf.Reference())
	return tr, err
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
