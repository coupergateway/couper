package runtime

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/backend"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/ratelimit"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
)

func NewBackend(ctx *hcl.EvalContext, body *hclsyntax.Body, log *logrus.Entry,
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
		return nil, errors.Configuration.Label(name).With(err)
	}

	// to prevent weird debug sessions; max to set the internal memStore ttl limit.
	store.Set(prefix+name, b, math.MaxInt64)

	return b.(http.RoundTripper), nil
}

func newBackend(evalCtx *hcl.EvalContext, backendCtx *hclsyntax.Body, log *logrus.Entry,
	conf *config.Couper, memStore *cache.MemoryStore) (http.RoundTripper, error) {
	beConf := &config.Backend{}
	if diags := gohcl.DecodeBody(backendCtx, evalCtx, beConf); diags.HasErrors() {
		return nil, diags
	}

	var err error
	if beConf.Name == "" {
		beConf.Name, err = getBackendName(evalCtx, backendCtx)
		if err != nil {
			return nil, err
		}
	}

	tc := &transport.Config{
		BackendName:            beConf.Name,
		Certificate:            conf.Settings.Certificate,
		Context:                conf.Context,
		DisableCertValidation:  beConf.DisableCertValidation,
		DisableConnectionReuse: beConf.DisableConnectionReuse,
		HTTP2:                  beConf.HTTP2,
		NoProxyFromEnv:         conf.Settings.NoProxyFromEnv,
		MaxConnections:         beConf.MaxConnections,
	}

	tc.CACertificate, tc.ClientCertificate, err = transport.ReadCertificates(beConf.TLS)
	if err != nil {
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  err.(errors.GoError).LogError(),
			Subject:  &backendCtx.SrcRange,
		}}
	}

	if len(beConf.RateLimits) > 0 {
		if strings.HasPrefix(beConf.Name, "anonymous_") {
			return nil, fmt.Errorf("anonymous backend '%s' cannot define 'beta_rate_limit' block(s)", beConf.Name)
		}

		tc.RateLimits, err = ratelimit.ConfigureRateLimits(conf.Context, beConf.RateLimits, log)
		if err != nil {
			return nil, err
		}
	}

	options := &transport.BackendOptions{}

	opts, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	if err != nil {
		return nil, err
	}

	options.OpenAPI = opts

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

	for _, blockType := range []string{
		config.OAuthBlockSchema.Blocks[0].Type,
		config.TokenRequestBlockSchema.Blocks[0].Type,
	} {
		blocks := hclbody.BlocksOfType(backendCtx, blockType)
		for _, block := range blocks {
			var requestAuthorizer transport.RequestAuthorizer
			requestAuthorizer, err = newRequestAuthorizer(evalCtx, block, log, conf, memStore)
			if err != nil {
				return nil, err
			}
			options.RequestAuthz = append(options.RequestAuthz, requestAuthorizer)
		}
	}

	b := transport.NewBackend(backendCtx, tc, options, log)
	return b, nil
}

func newRequestAuthorizer(evalCtx *hcl.EvalContext, block *hclsyntax.Block,
	log *logrus.Entry, conf *config.Couper, memStore *cache.MemoryStore) (transport.RequestAuthorizer, error) {
	var authorizerConfig interface{}
	switch block.Type {
	case config.OAuthBlockSchema.Blocks[0].Type:
		var one uint8 = 1
		authorizerConfig = &config.OAuth2ReqAuth{
			Retries: &one,
		}
	case config.TokenRequestBlockSchema.Blocks[0].Type:
		// block is guaranteed to have a label ("default" being added at configload)
		authorizerConfig = &config.TokenRequest{Name: block.Labels[0]}
	default:
		return nil, errors.Configuration.Messagef("request authorizer not implemented: %s", block.Type)
	}

	if diags := gohcl.DecodeBody(block.Body, evalCtx, authorizerConfig); diags.HasErrors() {
		return nil, diags
	}

	backendBlocks := hclbody.BlocksOfType(block.Body, "backend")
	if len(backendBlocks) == 0 {
		r := block.Body.SrcRange
		diag := &hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &r,
			Summary: "missing backend initialization",
		}}
		return nil, errors.Configuration.Label("unexpected").With(diag)
	}

	innerBackend := backendBlocks[0] // backend block is set by configload package
	authorizerBackend, err := NewBackend(evalCtx, innerBackend.Body, log, conf, memStore)
	if err != nil {
		return nil, err
	}

	switch impl := authorizerConfig.(type) {
	case *config.OAuth2ReqAuth:
		return transport.NewOAuth2ReqAuth(evalCtx, impl, memStore, authorizerBackend)
	case *config.TokenRequest:
		reqs := producer.Requests{&producer.Request{
			Backend: authorizerBackend,
			Context: impl.HCLBody(),
			Name:    impl.Name,
		}}
		return transport.NewTokenRequest(impl, memStore, reqs)
	default:
		return nil, errors.Configuration.Message("unknown authorizer type")
	}
}

func getBackendName(evalCtx *hcl.EvalContext, backendCtx *hclsyntax.Body) (string, error) {
	if n, exist := backendCtx.Attributes["name"]; exist {
		v, err := eval.Value(evalCtx, n.Expr)
		if err != nil {
			return "", err
		}

		return v.AsString(), nil
	}

	return "", nil
}
