package transport

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coupergateway/couper/config"
	hclbody "github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/eval/variables"
	"github.com/coupergateway/couper/handler/throttle"
	"github.com/coupergateway/couper/handler/validation"
	"github.com/coupergateway/couper/internal/seetie"
	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/server/writer"
	"github.com/coupergateway/couper/telemetry"
	"github.com/coupergateway/couper/utils"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ http.RoundTripper = &Backend{}
	_ ProbeStateChange  = &Backend{}
	_ seetie.Object     = &Backend{}
)

type Backend struct {
	context             *hclsyntax.Body
	healthInfo          *HealthInfo
	healthyMu           sync.RWMutex
	logEntry            *logrus.Entry
	name                string
	openAPIValidator    *validation.OpenAPI
	requestAuthorizer   []RequestAuthorizer
	transport           http.RoundTripper
	transportConf       *Config
	transportConfResult Config
	transportOnce       sync.Once
	upstreamLog         *logging.UpstreamLog
}

// NewBackend creates a new <*Backend> object by the given <*Config>.
func NewBackend(ctx *hclsyntax.Body, tc *Config, opts *BackendOptions, log *logrus.Entry) http.RoundTripper {
	var (
		healthCheck       *config.HealthCheck
		openAPI           *validation.OpenAPI
		requestAuthorizer []RequestAuthorizer
	)

	if opts != nil {
		healthCheck = opts.HealthCheck
		openAPI = validation.NewOpenAPI(opts.OpenAPI)
		requestAuthorizer = opts.RequestAuthz
	}

	backend := &Backend{
		context:           ctx,
		healthInfo:        &HealthInfo{Healthy: true, State: StateOk.String()},
		logEntry:          log.WithField("backend", tc.BackendName),
		name:              tc.BackendName,
		openAPIValidator:  openAPI,
		requestAuthorizer: requestAuthorizer,
		transportConf:     tc,
	}

	backend.upstreamLog = logging.NewUpstreamLog(backend.logEntry, backend, tc.NoProxyFromEnv)

	distinct := !strings.HasPrefix(tc.BackendName, "anonymous_")
	if distinct && healthCheck != nil {
		NewProbe(backend.logEntry, tc, healthCheck, backend)
	}

	return backend.upstreamLog
}

// initOnce ensures synced transport configuration. First request will setup the rate limits, origin, hostname and tls.
func (b *Backend) initOnce(conf *Config) {
	var innerTransport http.RoundTripper
	if len(b.transportConf.Throttles) > 0 {
		innerTransport = throttle.NewLimiter(NewTransport(conf, b.logEntry), b.transportConf.Throttles)
	} else {
		innerTransport = NewTransport(conf, b.logEntry)
	}
	b.transport = telemetry.NewInstrumentedRoundTripper(innerTransport)

	b.healthyMu.Lock()
	b.transportConfResult = *conf
	healthy := b.healthInfo.Healthy
	healthState := b.healthInfo.State
	b.healthyMu.Unlock()

	// race condition, update possible healthy backend with current origin and hostname
	b.OnProbeChange(&HealthInfo{Healthy: healthy, State: healthState})
}

// RoundTrip implements the <http.RoundTripper> interface.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	ctxBody, _ := req.Context().Value(request.BackendParams).(*hclsyntax.Body)
	if ctxBody == nil {
		ctxBody = b.context
	} else {
		ctxBody = hclbody.MergeBodies(ctxBody, b.context, false)
	}

	outreq := req.WithContext(context.WithValue(req.Context(), request.BackendName, b.name))

	// originalReq for token-request retry purposes
	originalReq, err := b.withTokenRequest(outreq)
	if err != nil {
		return nil, errors.BetaBackendTokenRequest.Label(b.name).With(err)
	}

	var backendVal cty.Value
	hclCtx := eval.ContextFromRequest(outreq).HCLContextSync()
	if v, ok := hclCtx.Variables[variables.Backends]; ok {
		if m, exist := v.AsValueMap()[b.name]; exist {
			hclCtx.Variables[variables.Backend] = m
			backendVal = m
		}
	}

	if err = b.isUnhealthy(hclCtx, ctxBody); err != nil {
		return &http.Response{
			Request: req, // provide outreq (variable) on error cases
		}, err
	}

	// Execute before <b.evalTransport()> due to right
	// handling of query-params in the URL attribute.
	if err = eval.ApplyRequestContext(hclCtx, ctxBody, outreq); err != nil {
		return nil, err
	}

	// TODO: split timing eval
	tc, err := b.evalTransport(hclCtx, ctxBody, outreq)
	if err != nil {
		return nil, err
	}

	// first traffic pins the origin settings to transportConfResult
	b.transportOnce.Do(func() {
		b.initOnce(tc)
	})

	// use result and apply context timings
	b.healthyMu.RLock()
	tconf := b.transportConfResult
	b.healthyMu.RUnlock()
	tconf.ConnectTimeout = tc.ConnectTimeout
	tconf.TTFBTimeout = tc.TTFBTimeout
	tconf.Timeout = tc.Timeout

	deadlineErr := b.withTimeout(outreq, &tconf)

	outreq.URL.Host = tconf.Origin
	outreq.URL.Scheme = tconf.Scheme
	outreq.Host = tconf.Hostname

	// handler.Proxy marks proxy round-trips since we should not handle headers twice.
	_, isProxyReq := outreq.Context().Value(request.RoundTripProxy).(bool)

	if !isProxyReq {
		RemoveConnectionHeaders(outreq.Header)
		RemoveHopHeaders(outreq.Header)
	}

	writer.ModifyAcceptEncoding(outreq.Header)

	if xff, ok := outreq.Context().Value(request.XFF).(string); ok {
		if xff != "" {
			outreq.Header.Set("X-Forwarded-For", xff)
		} else {
			outreq.Header.Del("X-Forwarded-For")
		}
	}

	b.withBasicAuth(outreq, hclCtx, ctxBody)
	if err = b.withPathPrefix(outreq, hclCtx, ctxBody); err != nil {
		return nil, err
	}

	setUserAgent(outreq)
	outreq.Close = false

	if _, ok := req.Context().Value(request.WebsocketsAllowed).(bool); !ok {
		outreq.Header.Del("Connection")
		outreq.Header.Del("Upgrade")
	}

	var beresp *http.Response
	if b.openAPIValidator != nil {
		beresp, err = b.openAPIValidate(outreq, &tconf, deadlineErr)
	} else {
		beresp, err = b.innerRoundTrip(outreq, &tconf, deadlineErr)
	}

	if err != nil {
		if beresp == nil {
			beresp = &http.Response{
				Request: outreq,
			} // provide outreq (variable) on error cases
		}
		if varSync, ok := outreq.Context().Value(request.ContextVariablesSynced).(*eval.SyncedVariables); ok {
			varSync.SetResp(beresp)
		}
		return beresp, err
	}

	if retry, rerr := b.withRetryTokenRequest(outreq, beresp); rerr != nil {
		return beresp, errors.BetaBackendTokenRequest.Label(b.name).With(rerr)
	} else if retry {
		return b.RoundTrip(originalReq)
	}

	if !eval.IsUpgradeResponse(outreq, beresp) {
		beresp.Body = logging.NewBytesCountReader(beresp)
		if err = setGzipReader(beresp); err != nil {
			b.upstreamLog.LogEntry().WithContext(outreq.Context()).WithError(err).Error()
		}
	}

	if !isProxyReq {
		RemoveConnectionHeaders(beresp.Header)
		RemoveHopHeaders(beresp.Header)
	}

	// Backend response context creates the beresp variables in first place and applies this context
	// to the current beresp obj. Downstream response context evals reading their beresp variable values
	// from this result.
	evalCtx := eval.ContextFromRequest(outreq)
	var bereqV, berespV cty.Value
	var reqName string
	evalCtx, reqName, bereqV, berespV = evalCtx.WithBeresp(beresp, backendVal)

	clfValue, err := eval.EvalCustomLogFields(evalCtx.HCLContext(), ctxBody)
	if err != nil {
		logError, _ := outreq.Context().Value(request.LogCustomUpstreamError).(*error)
		*logError = err
	} else if clfValue != cty.NilVal {
		logValue, _ := outreq.Context().Value(request.LogCustomUpstreamValue).(*cty.Value)
		*logValue = clfValue
	}

	err = eval.ApplyResponseContext(evalCtx.HCLContext(), ctxBody, beresp)

	if varSync, ok := outreq.Context().Value(request.ContextVariablesSynced).(*eval.SyncedVariables); ok {
		varSync.Set(outreq.Context(), reqName, bereqV, berespV)
	}

	return beresp, err
}

func (b *Backend) openAPIValidate(req *http.Request, tc *Config, deadlineErr <-chan error) (*http.Response, error) {
	requestValidationInput, err := b.openAPIValidator.ValidateRequest(req)
	if err != nil {
		return nil, errors.BackendOpenapiValidation.Label(b.name).With(err)
	}

	beresp, err := b.innerRoundTrip(req, tc, deadlineErr)
	if err != nil {
		return nil, err
	}

	if err = b.openAPIValidator.ValidateResponse(beresp, requestValidationInput); err != nil {
		return beresp, errors.BackendOpenapiValidation.Label(b.name).With(err).Status(http.StatusBadGateway)
	}

	return beresp, nil
}

func (b *Backend) innerRoundTrip(req *http.Request, tc *Config, deadlineErr <-chan error) (*http.Response, error) {
	beresp, err := b.transport.RoundTrip(req)

	if err != nil {
		select {
		case derr := <-deadlineErr:
			if derr != nil {
				return nil, derr
			}
		default:
			if _, ok := err.(*errors.Error); ok {
				return nil, err
			}

			return nil, errors.Backend.Label(b.name).With(err)
		}
	}

	return beresp, nil
}

func (b *Backend) withTokenRequest(req *http.Request) (*http.Request, error) {
	if b.requestAuthorizer == nil {
		return nil, nil
	}

	// Reset for upstream transport; prevent mixing values.
	// requestAuthorizer will have their own backend configuration.
	ctx := context.WithValue(req.Context(), request.BackendParams, nil)

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	originalReq := req.Clone(req.Context())

	// WithContext() instead of Clone() due to header-map modification.
	req = req.WithContext(ctx)

	errorsCh := make(chan error, len(b.requestAuthorizer))
	for _, authorizer := range b.requestAuthorizer {
		err := authorizer.GetToken(req)
		if err != nil {
			return originalReq, err
		}

		go func(ra RequestAuthorizer, r *http.Request) {
			errorsCh <- ra.GetToken(r)
		}(authorizer, req)
	}

	var err error
	for i := 0; i < len(b.requestAuthorizer); i++ {
		err = <-errorsCh
		if err != nil {
			break
		}
	}
	return originalReq, err
}

func (b *Backend) withRetryTokenRequest(req *http.Request, res *http.Response) (bool, error) {
	if len(b.requestAuthorizer) == 0 {
		return false, nil
	}

	var retry bool
	for _, ra := range b.requestAuthorizer {
		r, err := ra.RetryWithToken(req, res)
		if err != nil {
			return false, err
		}
		if r {
			retry = true
			break
		}
	}
	return retry, nil
}

func (b *Backend) withPathPrefix(req *http.Request, evalCtx *hcl.EvalContext, hclContext *hclsyntax.Body) error {
	if pathPrefix := b.getAttribute(evalCtx, "path_prefix", hclContext); pathPrefix != "" {
		if i := strings.Index(pathPrefix, "#"); i >= 0 {
			return errors.Configuration.Messagef("path_prefix attribute: invalid fragment found in %q", pathPrefix)
		} else if i = strings.Index(pathPrefix, "?"); i >= 0 {
			return errors.Configuration.Messagef("path_prefix attribute: invalid query string found in %q", pathPrefix)
		}

		if err := eval.ValidatePath(pathPrefix, "path_prefix attribute"); err != nil {
			return err
		}

		req.URL.Path = utils.JoinPath("/", pathPrefix, req.URL.Path)
	}

	return nil
}

func (b *Backend) withBasicAuth(req *http.Request, evalCtx *hcl.EvalContext, hclContext *hclsyntax.Body) {
	if creds := b.getAttribute(evalCtx, "basic_auth", hclContext); creds != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(creds))
		req.Header.Set("Authorization", "Basic "+auth)
	}
}

func (b *Backend) getAttribute(evalContext *hcl.EvalContext, name string, hclContext *hclsyntax.Body) string {
	attrVal, err := eval.ValueFromBodyAttribute(evalContext, hclContext, name)
	if err != nil {
		b.upstreamLog.LogEntry().WithError(errors.Evaluation.Label(b.name).With(err))
	}
	return seetie.ValueToString(attrVal)
}

func (b *Backend) withTimeout(req *http.Request, conf *Config) <-chan error {
	timeout := conf.Timeout
	ws := false
	if to, ok := req.Context().Value(request.WebsocketsTimeout).(time.Duration); ok {
		timeout = to
		ws = true
	}

	errCh := make(chan error, 1)
	if timeout+conf.TTFBTimeout <= 0 {
		return errCh
	}

	ctx, cancel := context.WithCancel(context.WithValue(req.Context(), request.ConnectTimeout, conf.ConnectTimeout))

	downstreamTrace := httptrace.ContextClientTrace(ctx) // e.g. log-timings

	ttfbTimeout := make(chan time.Time, 1) // size to always cleanup related go-routine
	ttfbTimer := time.NewTimer(conf.TTFBTimeout)
	ctxTrace := &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if downstreamTrace != nil && downstreamTrace.WroteRequest != nil {
				downstreamTrace.WroteRequest(info)
			}

			if conf.TTFBTimeout <= 0 {
				return
			}

			go func(c context.Context, timeoutCh chan time.Time) {
				ttfbTimer.Reset(conf.TTFBTimeout)
				select {
				case <-c.Done():
					ttfbTimer.Stop()
					select {
					case <-ttfbTimer.C:
					default:
					}
				case t := <-ttfbTimer.C:
					// buffered, no select done required
					timeoutCh <- t
				}
			}(ctx, ttfbTimeout)
		},
		GotFirstResponseByte: func() {
			if downstreamTrace != nil && downstreamTrace.GotFirstResponseByte != nil {
				downstreamTrace.GotFirstResponseByte()
			}
			ttfbTimer.Stop()
		},
	}

	*req = *req.WithContext(httptrace.WithClientTrace(ctx, ctxTrace))

	go func(c context.Context, cancelFn func(), ec chan error) {
		defer cancelFn()
		deadline := make(<-chan time.Time)
		if timeout > 0 {
			deadlineTimer := time.NewTimer(timeout)
			deadline = deadlineTimer.C
			defer deadlineTimer.Stop()
		}
		select {
		case <-deadline:
			if ws {
				ec <- errors.BackendTimeout.Label(b.name).Message("websockets: deadline exceeded")
				return
			}
			ec <- errors.BackendTimeout.Label(b.name).Message("deadline exceeded")
			return
		case <-ttfbTimeout:
			ec <- errors.BackendTimeout.Label(b.name).Message("timeout awaiting response headers")
		case <-c.Done():
			return
		}
	}(ctx, cancel, errCh)
	return errCh
}

func (b *Backend) evalTransport(httpCtx *hcl.EvalContext, params *hclsyntax.Body, req *http.Request) (*Config, error) {
	log := b.upstreamLog.LogEntry()

	var origin, hostname, proxyURL string
	var connectTimeout, ttfbTimeout, timeout string
	type pair struct {
		attrName string
		target   *string
	}

	for _, p := range []pair{
		{"origin", &origin},
		{"hostname", &hostname},
		{"proxy", &proxyURL},
		// dynamic timings
		{"connect_timeout", &connectTimeout},
		{"ttfb_timeout", &ttfbTimeout},
		{"timeout", &timeout},
	} {
		if v, err := eval.ValueFromBodyAttribute(httpCtx, params, p.attrName); err != nil {
			log.WithError(errors.Evaluation.Label(b.name).With(err)).Error()
		} else if v != cty.NilVal {
			*p.target = seetie.ValueToString(v)
		}
	}

	originURL, parseErr := url.Parse(origin)
	if parseErr != nil {
		return nil, errors.Configuration.Label(b.name).With(parseErr)
	} else if strings.HasPrefix(originURL.Host, originURL.Scheme+":") {
		return nil, errors.Configuration.Label(b.name).
			Messagef("invalid url: %s", originURL.String())
	} else if origin == "" {
		// For internal backends (e.g. JWKS, OIDC) where the request URL is the
		// full absolute URL, derive origin from it. For user-facing proxy/request
		// backends (relative URL), this is a configuration error.
		if req.URL.IsAbs() && req.URL.Hostname() != "" {
			originURL = req.URL
		} else {
			return nil, errors.Configuration.Label(b.name).
				Message("the origin attribute must not be empty")
		}
	}

	if hostname == "" {
		hostname = originURL.Host
	}

	if !originURL.IsAbs() || originURL.Hostname() == "" {
		return nil, errors.Configuration.Label(b.name).
			Messagef("the origin attribute has to contain an absolute URL with a valid hostname: %q", origin)
	}

	return b.transportConf.
		WithTarget(originURL.Scheme, originURL.Host, hostname, proxyURL).
		WithTimings(connectTimeout, ttfbTimeout, timeout, log), nil
}

func (b *Backend) isUnhealthy(ctx *hcl.EvalContext, params *hclsyntax.Body) error {
	val, err := eval.ValueFromBodyAttribute(ctx, params, "use_when_unhealthy")
	if err != nil {
		return err
	}

	var useUnhealthy bool
	if val.Type() == cty.Bool {
		useUnhealthy = val.True()
	} // else not set

	b.healthyMu.RLock()
	defer b.healthyMu.RUnlock()

	if b.healthInfo.Healthy || useUnhealthy {
		return nil
	}

	return errors.BackendUnhealthy
}

func (b *Backend) OnProbeChange(info *HealthInfo) {
	b.healthyMu.Lock()
	b.healthInfo = info
	b.healthyMu.Unlock()
}

func (b *Backend) Value() cty.Value {
	b.healthyMu.RLock()
	defer b.healthyMu.RUnlock()

	var tokens map[string]interface{}
	for _, auth := range b.requestAuthorizer {
		if name, v := auth.value(); v != "" {
			if tokens == nil {
				tokens = make(map[string]interface{})
			}
			tokens[name] = v
		}
	}

	result := map[string]interface{}{
		"health": map[string]interface{}{
			"healthy": b.healthInfo.Healthy,
			"error":   b.healthInfo.Error,
			"state":   b.healthInfo.State,
		},
		"hostname":        b.transportConfResult.Hostname,
		"name":            b.name, // mandatory
		"origin":          b.transportConfResult.Origin,
		"connect_timeout": b.transportConfResult.ConnectTimeout.String(),
		"ttfb_timeout":    b.transportConfResult.TTFBTimeout.String(),
		"timeout":         b.transportConfResult.Timeout.String(),
	}

	if tokens != nil {
		result["beta_tokens"] = tokens
		if token, ok := tokens["default"]; ok {
			result["beta_token"] = token
		}
	}

	return seetie.GoToValue(result)
}

// setUserAgent sets an empty one if none is present or empty
// to prevent the go http defaultUA gets written.
func setUserAgent(outreq *http.Request) {
	if ua := outreq.Header.Get("User-Agent"); ua == "" {
		outreq.Header.Set("User-Agent", "")
	}
}

// setGzipReader will set the gzip.Reader for Content-Encoding gzip.
// Invalid header reads will reset the response.Body and return the related error.
func setGzipReader(beresp *http.Response) error {
	if strings.ToLower(beresp.Header.Get(writer.ContentEncodingHeader)) != writer.GzipName {
		return nil
	}

	bufOpt := beresp.Request.Context().Value(request.BufferOptions).(buffer.Option)
	if !bufOpt.Response() {
		return nil
	}

	var src io.Reader
	src, err := gzip.NewReader(beresp.Body)
	if err != nil {
		return errors.Backend.With(err).Message("body reset")
	}

	beresp.Header.Del(writer.ContentEncodingHeader)
	beresp.Header.Del("Content-Length")
	beresp.Body = eval.NewReadCloser(src, beresp.Body)
	return nil
}

// RemoveConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func RemoveConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func RemoveHopHeaders(header http.Header) {
	for _, h := range HopHeaders {
		hv := header.Get(h)
		if hv == "" {
			continue
		}
		if h == "Te" && hv == "trailers" {
			// Issue 21096: tell backend applications that
			// care about trailer support that we support
			// trailers. (We do, but we don't go out of
			// our way to advertise that unless the
			// incoming client request thought it was
			// worth mentioning)
			continue
		}
		header.Del(h)
	}
}

// HopHeaders Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var HopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}
