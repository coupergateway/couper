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
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/server/writer"
	"github.com/avenga/couper/telemetry"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
	"github.com/avenga/couper/utils"
)

var _ http.RoundTripper = &Backend{}

// Backend represents the transport configuration.
type Backend struct {
	context          hcl.Body
	logEntry         *logrus.Entry
	name             string
	openAPIValidator *validation.OpenAPI
	tokenRequest     TokenRequest
	transportConf    *Config
	upstreamLog      *logging.UpstreamLog
}

// NewBackend creates a new <*Backend> object by the given <*Config>.
func NewBackend(ctx hcl.Body, tc *Config, opts *BackendOptions, log *logrus.Entry) http.RoundTripper {
	var logEntry *logrus.Entry

	if tc.BackendName != "" {
		logEntry = log.WithField("backend", tc.BackendName)
	} else {
		logEntry = log.WithField("backend", "default")
	}

	var openAPI *validation.OpenAPI
	var tr TokenRequest
	if opts != nil {
		openAPI = validation.NewOpenAPI(opts.OpenAPI)
		tr = opts.AuthBackend
	}

	backend := &Backend{
		context:          ctx,
		logEntry:         logEntry,
		name:             tc.BackendName,
		openAPIValidator: openAPI,
		tokenRequest:     tr,
		transportConf:    tc,
	}
	backend.upstreamLog = logging.NewUpstreamLog(logEntry, backend, tc.NoProxyFromEnv)
	return backend.upstreamLog
}

// RoundTrip implements the <http.RoundTripper> interface.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	// for token-request retry purposes
	originalReq := req.Clone(req.Context())

	if err := b.withTokenRequest(req); err != nil {
		return nil, err
	}

	ctxBody, _ := req.Context().Value(request.BackendParams).(hcl.Body)
	if ctxBody == nil {
		ctxBody = b.context
	} else {
		ctxBody = hclbody.MergeBodies(b.context, ctxBody)
	}

	logCh, _ := req.Context().Value(request.LogCustomUpstream).(chan hcl.Body)
	if logCh != nil {
		logCh <- ctxBody
	}

	hclCtx := eval.ContextFromRequest(req).HCLContextSync()

	// Execute before <b.evalTransport()> due to right
	// handling of query-params in the URL attribute.
	if err := eval.ApplyRequestContext(hclCtx, ctxBody, req); err != nil {
		return nil, err
	}

	tc, err := b.evalTransport(hclCtx, ctxBody, req)
	if err != nil {
		return nil, err
	}

	deadlineErr := b.withTimeout(req, tc)

	req.URL.Host = tc.Origin
	req.URL.Scheme = tc.Scheme
	req.Host = tc.Hostname

	// handler.Proxy marks proxy roundtrips since we should not handle headers twice.
	_, isProxyReq := req.Context().Value(request.RoundTripProxy).(bool)

	if !isProxyReq {
		RemoveConnectionHeaders(req.Header)
		RemoveHopHeaders(req.Header)
	}

	writer.ModifyAcceptEncoding(req.Header)

	if xff, ok := req.Context().Value(request.XFF).(string); ok {
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		} else {
			req.Header.Del("X-Forwarded-For")
		}
	}

	b.withBasicAuth(req, ctxBody)
	if err = b.withPathPrefix(req, ctxBody); err != nil {
		return nil, err
	}

	setUserAgent(req)
	req.Close = false

	if _, ok := req.Context().Value(request.WebsocketsAllowed).(bool); !ok {
		req.Header.Del("Connection")
		req.Header.Del("Upgrade")
	}

	var beresp *http.Response
	if b.openAPIValidator != nil {
		beresp, err = b.openAPIValidate(req, tc, deadlineErr)
	} else {
		beresp, err = b.innerRoundTrip(req, tc, deadlineErr)
	}

	if err != nil {
		berespErr := &http.Response{
			Request: req,
		} // provide outreq (variable) on error cases
		if varSync, ok := req.Context().Value(request.ContextVariablesSynced).(*eval.SyncedVariables); ok {
			varSync.Set(berespErr)
		}
		return berespErr, err
	}

	if retry, rerr := b.withRetryTokenRequest(req, beresp); rerr != nil {
		return beresp, rerr
	} else if retry {
		return b.RoundTrip(originalReq)
	}

	if !eval.IsUpgradeResponse(req, beresp) {
		if err = setGzipReader(beresp); err != nil {
			b.upstreamLog.LogEntry().WithContext(req.Context()).WithError(err).Error()
		}
	}

	if !isProxyReq {
		RemoveConnectionHeaders(beresp.Header)
		RemoveHopHeaders(beresp.Header)
	}

	// Backend response context creates the beresp variables in first place and applies this context
	// to the current beresp obj. Downstream response context evals reading their beresp variable values
	// from this result.
	evalCtx := eval.ContextFromRequest(req)
	// has own body variable reference?
	readBody := eval.MustBuffer(b.context)&eval.BufferResponse == eval.BufferResponse
	evalCtx = evalCtx.WithBeresp(beresp, readBody)
	err = eval.ApplyResponseContext(evalCtx.HCLContext(), ctxBody, beresp)

	if varSync, ok := req.Context().Value(request.ContextVariablesSynced).(*eval.SyncedVariables); ok {
		varSync.Set(beresp)
	}

	return beresp, err
}

func (b *Backend) openAPIValidate(req *http.Request, tc *Config, deadlineErr <-chan error) (*http.Response, error) {
	requestValidationInput, err := b.openAPIValidator.ValidateRequest(req, tc.hash())
	if err != nil {
		return nil, errors.BackendValidation.Label(b.name).Kind("backend_request_validation").With(err)
	}

	beresp, err := b.innerRoundTrip(req, tc, deadlineErr)
	if err != nil {
		return nil, err
	}

	if err = b.openAPIValidator.ValidateResponse(beresp, requestValidationInput); err != nil {
		return nil, errors.BackendValidation.Label(b.name).Kind("backend_response_validation").
			With(err).Status(http.StatusBadGateway)
	}

	return beresp, nil
}

func (b *Backend) innerRoundTrip(req *http.Request, tc *Config, deadlineErr <-chan error) (*http.Response, error) {
	span := trace.SpanFromContext(req.Context())
	span.SetAttributes(telemetry.KeyOrigin.String(tc.Origin))
	span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(req)...)

	spanMsg := "backend"
	if b.name != "" {
		spanMsg += "." + b.name
	}

	meter := provider.Meter("couper/backend")
	counter := metric.Must(meter).NewInt64Counter(instrumentation.BackendRequest, metric.WithDescription(string(unit.Dimensionless)))
	duration := metric.Must(meter).
		NewFloat64Histogram(instrumentation.BackendRequestDuration, metric.WithDescription(string(unit.Dimensionless)))
	attrs := []attribute.KeyValue{
		attribute.String("backend_name", tc.BackendName),
		attribute.String("hostname", tc.Hostname),
		attribute.String("method", req.Method),
		attribute.String("origin", tc.Origin),
	}

	t := Get(tc, b.logEntry)
	start := time.Now()
	span.AddEvent(spanMsg + ".request")
	beresp, err := t.RoundTrip(req)
	span.AddEvent(spanMsg + ".response")
	endSeconds := time.Since(start).Seconds()

	statusKey := attribute.Key("code")
	if err != nil {
		defer meter.RecordBatch(req.Context(),
			append(attrs, statusKey.Int(0)),
			counter.Measurement(1),
			duration.Measurement(endSeconds))
		select {
		case derr := <-deadlineErr:
			if derr != nil {
				return nil, derr
			}
		default:
			return nil, errors.Backend.Label(b.name).With(err)
		}
	}

	meter.RecordBatch(req.Context(),
		append(attrs, statusKey.Int(beresp.StatusCode)),
		counter.Measurement(1),
		duration.Measurement(endSeconds))

	return beresp, nil
}

const backendTokenRequest = "backendTokenRequest"

func (b *Backend) withTokenRequest(req *http.Request) error {
	if b.tokenRequest == nil {
		return nil
	}

	trValue, _ := req.Context().Value(backendTokenRequest).(string)
	if trValue != "" { // prevent loop
		return nil
	}

	// prevent mixing context values - start from scratch
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, backendTokenRequest, "tr")

	go func(req *http.Request, cancelFn func()) {
		defer cancelFn()
		select {
		case <-req.Context().Done():
		}
	}(req, cancel)

	return b.tokenRequest.WithToken(req.WithContext(ctx))
}

func (b *Backend) withRetryTokenRequest(req *http.Request, res *http.Response) (bool, error) {
	if b.tokenRequest == nil {
		return false, nil
	}

	trValue, _ := req.Context().Value(backendTokenRequest).(string)
	if trValue != "" { // prevent loop
		return false, nil
	}

	return b.tokenRequest.RetryWithToken(req, res)
}

func (b *Backend) withPathPrefix(req *http.Request, hclContext hcl.Body) error {
	if pathPrefix := b.getAttribute(req, "path_prefix", hclContext); pathPrefix != "" {
		// TODO: Check for a valid absolute path
		if i := strings.Index(pathPrefix, "#"); i >= 0 {
			return errors.Configuration.Messagef("path_prefix attribute: invalid fragment found in \"%s\"", pathPrefix)
		} else if i := strings.Index(pathPrefix, "?"); i >= 0 {
			return errors.Configuration.Messagef("path_prefix attribute: invalid query string found in \"%s\"", pathPrefix)
		}

		req.URL.Path = utils.JoinPath("/", pathPrefix, req.URL.Path)
	}

	return nil
}

func (b *Backend) withBasicAuth(req *http.Request, hclContext hcl.Body) {
	if creds := b.getAttribute(req, "basic_auth", hclContext); creds != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(creds))
		req.Header.Set("Authorization", "Basic "+auth)
	}
}

func (b *Backend) getAttribute(req *http.Request, name string, hclContext hcl.Body) string {
	attrVal, err := eval.ValueFromBodyAttribute(eval.ContextFromRequest(req).HCLContext(), hclContext, name)
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

	ttfbTimeout := make(chan time.Time, 1) // size to always cleanup related go-routine
	ttfbInTime := make(chan struct{})
	ctxTrace := &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if conf.TTFBTimeout <= 0 {
				return
			}
			go func() {
				ttfbTimeout <- <-time.After(conf.TTFBTimeout)
			}()
		},
		GotFirstResponseByte: func() {
			close(ttfbInTime)
		},
	}

	*req = *req.WithContext(httptrace.WithClientTrace(ctx, ctxTrace))

	go func(cancelFn func(), c context.Context, ec chan error) {
		defer cancelFn()
		deadline := make(<-chan time.Time)
		if timeout > 0 {
			deadline = time.After(timeout)
		}
		select {
		case <-deadline:
			if ws {
				ec <- errors.BackendTimeout.Label(b.name).Message("websockets: deadline exceeded")
			} else {
				ec <- errors.BackendTimeout.Label(b.name).Message("deadline exceeded")
			}
			return
		case <-ttfbTimeout:
			select {
			case <-ttfbInTime: // last 'minute' call
			default:
				ec <- errors.BackendTimeout.Label(b.name).Message("timeout awaiting response headers")
			}
		case <-c.Done():
			return
		}
	}(cancel, ctx, errCh)
	return errCh
}

func (b *Backend) evalTransport(httpCtx *hcl.EvalContext, params hcl.Body, req *http.Request) (*Config, error) {
	log := b.upstreamLog.LogEntry()

	bodyContent, _, diags := params.PartialContent(config.BackendInlineSchema)
	if diags.HasErrors() {
		return nil, errors.Evaluation.Label(b.name).With(diags)
	}

	var origin, hostname, proxyURL, backendURL string
	var connectTimeout, ttfbTimeout, timeout string
	type pair struct {
		attrName string
		target   *string
	}
	for _, p := range []pair{
		{"origin", &origin},
		{"hostname", &hostname},
		{"proxy", &proxyURL},
		{"_backend_url", &backendURL}, // prepared by config-load
		// dynamic timings
		{"connect_timeout", &connectTimeout},
		{"ttfb_timeout", &ttfbTimeout},
		{"timeout", &timeout},
	} {
		if v, err := eval.ValueFromAttribute(httpCtx, bodyContent, p.attrName); err != nil {
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
	}

	// TODO: jwks_uri initialization
	if backendURL == "" {
		backendURL, _ = req.Context().Value(request.URLAttribute).(string)
	}

	if backendURL != "" {
		urlAttr, err := url.Parse(backendURL)
		if err != nil {
			return nil, errors.Configuration.Label(b.name).With(err)
		}

		if origin != "" && urlAttr.Host != originURL.Host {
			errctx := "url"
			if tr := req.Context().Value(request.TokenRequest); tr != nil {
				errctx = "token_endpoint"
			}
			return nil, errors.Configuration.Label(b.name).Kind(errctx).
				Messagef("backend: the host '%s' must be equal to 'backend.origin' host: '%s'",
					urlAttr.Host, originURL.Host)
		}

		originURL.Host = urlAttr.Host
		originURL.Scheme = urlAttr.Scheme
		req.URL.Scheme = urlAttr.Scheme

		if urlAttr.Path != "" {
			req.URL.Path = urlAttr.Path
		}

		if urlAttr.RawQuery != "" {
			req.URL.RawQuery = urlAttr.RawQuery
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
		WithTimings(connectTimeout, ttfbTimeout, timeout), nil
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

	bufOpt := beresp.Request.Context().Value(request.BufferOptions).(eval.BufferOption)
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
