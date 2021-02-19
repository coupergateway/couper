package transport

import (
	"compress/gzip"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"

	couperErr "github.com/avenga/couper/errors"

	"github.com/avenga/couper/handler/validation"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/logging"
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

// Backend represents the transport <Backend> object.
type Backend struct {
	accessControl    string // maps to basic-auth atm
	context          hcl.Body
	evalContext      *hcl.EvalContext
	name             string
	transportConf    *Config
	upstreamLog      *logging.AccessLog
	openAPIValidator *validation.OpenAPI
	// oauth
	// ...
	// TODO: OrderedList for origin AC, middlewares etc.
}

// NewBackend creates a new <*Backend> object by the given <*Config>.
func NewBackend(evalCtx *hcl.EvalContext, conf *Config, openAPIopts *validation.OpenAPIOptions) *Backend {
	return &Backend{
		evalContext:      evalCtx,
		openAPIValidator: validation.NewOpenAPI(openAPIopts),
		transportConf:    conf,
	}
}

// RoundTrip implements the <http.RoundTripper> interface.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	//startTime := time.Now()

	t := Get(b.transportConf)
	err := eval.ApplyRequestContext(b.evalContext, b.context, req)
	if err != nil {
		return nil, err
	}

	// oauth ....

	if b.accessControl != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(b.accessControl))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	removeConnectionHeaders(req.Header)
	removeHopHeaders(req.Header)

	if ReClientSupportsGZ.MatchString(req.Header.Get(AcceptEncodingHeader)) {
		req.Header.Set(AcceptEncodingHeader, GzipName)
	} else {
		req.Header.Del(AcceptEncodingHeader)
	}

	// TODO: logging, roundtrip
	roundtripInfo := &logging.RoundtripInfo{}

	if b.openAPIValidator != nil {
		if err = b.openAPIValidator.ValidateRequest(req, roundtripInfo); err != nil {
			//p.options.ErrorTemplate.ServeError(couperErr.UpstreamRequestValidationFailed).ServeHTTP(rw, req)
			return nil, couperErr.UpstreamRequestValidationFailed
		}
	}

	//b.upstreamLog.ServeHTTP(rw, req, logging.RoundtripHandlerFunc(t.RoundTrip), startTime)
	beresp, err := t.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if b.openAPIValidator != nil {
		if err = b.openAPIValidator.ValidateResponse(beresp, roundtripInfo); err != nil {
			return nil, couperErr.UpstreamResponseValidationFailed
		}
	}

	if strings.ToLower(beresp.Header.Get(ContentEncodingHeader)) == GzipName {
		src, rerr := gzip.NewReader(beresp.Body)
		if rerr == nil {
			beresp.Header.Del(ContentEncodingHeader)
			beresp.Body = eval.NewReadCloser(src, beresp.Body)
		}
	}

	removeConnectionHeaders(req.Header)

	err = eval.ApplyResponseContext(b.evalContext, b.context, req, beresp)
	return beresp, err
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
