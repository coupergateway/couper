package transport

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/utils"
)

const (
	GzipName              = "gzip"
	AcceptEncodingHeader  = "Accept-Encoding"
	ContentEncodingHeader = "Content-Encoding"
	ContentLengthHeader   = "Content-Length"
	VaryHeader            = "Vary"
)

var _ http.RoundTripper = &Backend{}

var ReClientSupportsGZ = regexp.MustCompile(`(?i)\b` + GzipName + `\b`)

// Backend represents the transport configuration.
type Backend struct {
	context          hcl.Body
	name             string
	openAPIValidator *validation.OpenAPI
	options          *BackendOptions
	transportConf    *Config
	upstreamLog      *logging.UpstreamLog
	// TODO: OrderedList for origin AC, middlewares etc.
}

// NewBackend creates a new <*Backend> object by the given <*Config>.
func NewBackend(ctx hcl.Body, tc *Config, opts *BackendOptions, log *logrus.Entry) http.RoundTripper {
	logEntry := log
	if tc.BackendName != "" {
		logEntry = log.WithField("backend", tc.BackendName)
	} else {
		logEntry = log.WithField("backend", "default")
	}

	var openAPI *validation.OpenAPI
	if opts != nil {
		openAPI = validation.NewOpenAPI(opts.OpenAPI)
	}

	backend := &Backend{
		context:          ctx,
		openAPIValidator: openAPI,
		options:          opts,
		transportConf:    tc,
	}
	backend.upstreamLog = logging.NewUpstreamLog(logEntry, backend, tc.NoProxyFromEnv)
	return backend.upstreamLog
}

// RoundTrip implements the <http.RoundTripper> interface.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	// Execute before <b.evalTransport()> due to right
	// handling of query-params in the URL attribute.
	err := eval.ApplyRequestContext(req.Context(), b.context, req)
	if err != nil {
		return nil, err
	}

	tc, err := b.evalTransport(req)
	if err != nil {
		return nil, err
	}

	t := Get(tc)

	deadlineErr := b.withTimeout(req)

	req.URL.Host = tc.Origin
	req.URL.Scheme = tc.Scheme
	req.Host = tc.Hostname

	// handler.Proxy marks proxy roundtrips since we should not handle headers twice.
	_, isProxyReq := req.Context().Value(request.RoundTripProxy).(bool)

	if !isProxyReq {
		removeConnectionHeaders(req.Header)
		removeHopHeaders(req.Header)
	}

	if ReClientSupportsGZ.MatchString(req.Header.Get(AcceptEncodingHeader)) {
		req.Header.Set(AcceptEncodingHeader, GzipName)
	} else {
		req.Header.Del(AcceptEncodingHeader)
	}

	if b.openAPIValidator != nil {
		if err = b.openAPIValidator.ValidateRequest(req); err != nil {
			return nil, errors.Backend.Label(b.name).With(err).Message("request validation failed")
		}
	}

	if xff, ok := req.Context().Value(request.XFF).(string); ok {
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		} else {
			req.Header.Del("X-Forwarded-For")
		}
	}

	b.withBasicAuth(req)
	b.withPathPrefix(req)

	setUserAgent(req)
	req.Close = false

	ctx := context.WithValue(req.Context(), request.BackendURL, req.URL.String())
	*req = *req.WithContext(ctx)

	beresp, err := t.RoundTrip(req)
	if err != nil {
		select {
		case derr := <-deadlineErr:
			if derr != nil {
				return nil, derr
			}
		default:
			return nil, err
		}
	}

	if b.openAPIValidator != nil {
		if err = b.openAPIValidator.ValidateResponse(beresp); err != nil {
			return nil, errors.Backend.Label(b.name).With(err).Message("response validation failed")
		}
	}

	if strings.ToLower(beresp.Header.Get(ContentEncodingHeader)) == GzipName {
		src, rerr := gzip.NewReader(beresp.Body)
		if rerr == nil {
			beresp.Header.Del(ContentEncodingHeader)
			beresp.Body = eval.NewReadCloser(src, beresp.Body)
		}
	}

	if !isProxyReq {
		removeConnectionHeaders(req.Header)
	}

	// Backend response context creates the beresp variables in first place and applies this context
	// to the current beresp obj. Downstream response context evals reading their beresp variable values
	// from this result. Fallback case is for testing purposes.
	if evalCtx, ok := req.Context().Value(eval.ContextType).(*eval.Context); ok {
		evalCtx = evalCtx.WithBeresps(beresp)
		err = eval.ApplyResponseContext(evalCtx, b.context, beresp)
	} else {
		err = eval.ApplyResponseContext(req.Context(), b.context, beresp)
	}

	return beresp, err
}

func (b *Backend) withPathPrefix(req *http.Request) {
	if pathPrefix := b.getAttribute(req, "path_prefix"); pathPrefix != "" {
		req.URL.Path = utils.JoinPath("/", pathPrefix, req.URL.Path)
	}
}

func (b *Backend) withBasicAuth(req *http.Request) {
	if creds := b.getAttribute(req, "basic_auth"); creds != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(creds))
		req.Header.Set("Authorization", "Basic "+auth)
	}
}

func (b *Backend) getAttribute(req *http.Request, name string) string {
	attrVal, err := eval.GetContextAttribute(b.context, req.Context(), name)
	if err != nil {
		b.upstreamLog.LogEntry().WithField("hcl", "backend").Error(err)
	}
	return attrVal
}

func (b *Backend) withTimeout(req *http.Request) <-chan error {
	errCh := make(chan error, 1)
	if b.transportConf.Timeout <= 0 {
		return errCh
	}

	ctx, cancel := context.WithCancel(req.Context())
	*req = *req.WithContext(ctx)
	go func(cancelFn func(), c context.Context, ec chan error) {
		defer cancelFn()
		deadline := time.After(b.transportConf.Timeout)
		select {
		case <-deadline:
			ec <- errors.Timeout.Label(b.name).Message("deadline exceeded")
			return
		case <-c.Done():
			return
		}
	}(cancel, ctx, errCh)
	return errCh
}

func (b *Backend) evalTransport(req *http.Request) (*Config, error) {
	var httpContext *hcl.EvalContext
	if httpCtx, ok := req.Context().Value(eval.ContextType).(*eval.Context); ok {
		httpContext = httpCtx.HCLContext()
	}

	log := b.upstreamLog.LogEntry().WithField("hcl", "backend")

	content, _, diags := b.context.PartialContent(config.BackendInlineSchema)
	if diags.HasErrors() {
		log.Error(diags)
	}

	var origin, hostname, proxyURL string
	type pair struct {
		attrName string
		target   *string
	}
	for _, p := range []pair{
		{"origin", &origin},
		{"hostname", &hostname},
		{"proxy", &proxyURL},
	} {
		if v, err := eval.GetAttribute(httpContext, content, p.attrName); err != nil {
			log.Error(err)
		} else if v != "" {
			*p.target = v
		}
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		log.Error(err)
	}

	if rawURL, ok := req.Context().Value(request.URLAttribute).(string); ok {
		urlAttr, err := url.Parse(rawURL)
		if err != nil {
			log.Error(err)
		}

		if origin != "" && urlAttr.Host != originURL.Host {
			errctx := "url"
			if tr := req.Context().Value(request.TokenRequest); tr != nil {
				errctx = "token_endpoint"
			}
			return nil, fmt.Errorf(
				"backend: the host of '%s': %q and 'backend.origin': %q must be equal",
				errctx, urlAttr.Host, origin)
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
		return nil, fmt.Errorf("the origin attribute has to contain an absolute URL with a valid hostname: %q", origin)
	}

	return b.transportConf.With(originURL.Scheme, originURL.Host, hostname, proxyURL), nil
}

// setUserAgent sets an empty one if none is present or empty
// to prevent the go http defaultUA gets written.
func setUserAgent(outreq *http.Request) {
	if ua := outreq.Header.Get("User-Agent"); ua == "" {
		outreq.Header.Set("User-Agent", "")
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func removeHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
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

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
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
