package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpguts"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime/server"
	couperErr "github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
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

var (
	_ http.Handler   = &Proxy{}
	_ server.Context = &Proxy{}

	transports sync.Map

	// headerBlacklist lists all header keys which will be removed after
	// context variable evaluation to ensure to not pass them upstream.
	headerBlacklist = []string{"Authorization", "Cookie"}

	ReClientSupportsGZ = regexp.MustCompile(`(?i)\b` + GzipName + `\b`)

	backendInlineSchema = config.Backend{}.Schema(true)
)

type Proxy struct {
	bufferOption eval.BufferOption
	evalContext  *hcl.EvalContext
	log          *logrus.Entry
	options      *ProxyOptions
	optionsHash  string
	srvOptions   *server.Options
	transport    *http.Transport
	upstreamLog  *logging.AccessLog
}

type CORSOptions struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxAge           string
}

func NewCORSOptions(cors *config.CORS) (*CORSOptions, error) {
	if cors == nil {
		return nil, nil
	}
	dur, err := time.ParseDuration(cors.MaxAge)
	if err != nil {
		return nil, err
	}
	corsMaxAge := strconv.Itoa(int(math.Floor(dur.Seconds())))

	allowedOrigins := seetie.ValueToStringSlice(cors.AllowedOrigins)
	for i, a := range allowedOrigins {
		allowedOrigins[i] = strings.ToLower(a)
	}

	return &CORSOptions{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: cors.AllowCredentials,
		MaxAge:           corsMaxAge,
	}, nil
}

// NeedsVary if a request with not allowed origin is ignored.
func (c *CORSOptions) NeedsVary() bool {
	return !c.AllowsOrigin("*")
}

func (c *CORSOptions) AllowsOrigin(origin string) bool {
	if c == nil {
		return false
	}

	for _, a := range c.AllowedOrigins {
		if a == strings.ToLower(origin) || a == "*" {
			return true
		}
	}

	return false
}

func NewProxy(options *ProxyOptions, log *logrus.Entry, srvOpts *server.Options, evalCtx *hcl.EvalContext) (http.Handler, error) {
	logConf := *logging.DefaultConfig
	logConf.TypeFieldKey = "couper_backend"
	logConf.NoProxyFromEnv = options.NoProxyFromEnv
	env.DecodeWithPrefix(&logConf, "BACKEND_")

	var apiValidation eval.BufferOption
	if options.OpenAPI != nil {
		apiValidation = options.OpenAPI.buffer
	}

	proxy := &Proxy{
		bufferOption: apiValidation | eval.MustBuffer(options.Context),
		evalContext:  evalCtx,
		log:          log,
		options:      options,
		optionsHash:  options.Hash(),
		srvOptions:   srvOpts,
		upstreamLog:  logging.NewAccessLog(&logConf, log.Logger),
	}

	return proxy, nil
}

func (p *Proxy) getTransport(scheme, origin, hostname string) *http.Transport {
	key := scheme + "|" + origin + "|" + hostname + "|" + p.optionsHash
	transport, ok := transports.Load(key)
	if !ok {
		tlsConf := &tls.Config{
			InsecureSkipVerify: p.options.DisableCertValidation,
		}
		if origin != hostname {
			tlsConf.ServerName = hostname
		}

		d := &net.Dialer{Timeout: p.options.ConnectTimeout}

		var proxyFunc func(req *http.Request) (*url.URL, error)
		if p.options.Proxy != "" {

		} else if !p.options.NoProxyFromEnv {
			proxyFunc = http.ProxyFromEnvironment
		}

		var nextProto map[string]func(authority string, c *tls.Conn) http.RoundTripper
		if !p.options.HTTP2 {
			nextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s %q failed: %w", p.options.BackendName, addr, err)
				}
				return conn, nil
			},
			Dial: (&net.Dialer{
				KeepAlive: 60 * time.Second,
			}).Dial,
			DisableCompression:    true,
			DisableKeepAlives:     p.options.DisableConnectionReuse,
			MaxConnsPerHost:       p.options.MaxConnections,
			Proxy:                 proxyFunc,
			ResponseHeaderTimeout: p.options.TTFBTimeout,
			TLSClientConfig:       tlsConf,
			TLSNextProto:          nextProto,
		}

		transports.Store(key, transport)
	}
	if t, ok := transport.(*http.Transport); ok {
		return t
	}
	return nil
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	if isCorsPreflightRequest(req) {
		p.setCorsRespHeaders(rw.Header(), req)
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	*req = *req.Clone(context.WithValue(req.Context(), request.BackendName, p.options.BackendName))
	p.upstreamLog.ServeHTTP(rw, req, logging.RoundtripHandlerFunc(p.roundtrip), startTime)
}

func (p *Proxy) roundtrip(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if p.options.Timeout > 0 {
		deadline := time.Now().Add(p.options.Timeout)
		c, cancelFn := context.WithDeadline(req.Context(), deadline)
		ctx = c
		defer cancelFn()
	}

	outreq := req.Clone(ctx)
	if req.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	if outreq.Header == nil {
		outreq.Header = make(http.Header) // Issue 33142: historical behavior was to always allocate
	}

	roundtripInfo := req.Context().Value(request.RoundtripInfo).(*logging.RoundtripInfo)

	err := p.Director(outreq)
	if err != nil {
		roundtripInfo.Err = err
		p.options.ErrorTemplate.ServeError(err).ServeHTTP(rw, req)
		return
	}

	outreq.Close = false

	// Deal with req.post access on the way back
	if outreq.GetBody != nil {
		req.GetBody = outreq.GetBody
	}

	reqUpType := upgradeType(outreq.Header)
	removeConnectionHeaders(outreq.Header)

	removeHopHeaders(outreq.Header)

	// After stripping all the hop-by-hop connection headers above, add back any
	// necessary for protocol upgrades, such as for websockets.
	if reqUpType != "" {
		outreq.Header.Set("Connection", "Upgrade")
		outreq.Header.Set("Upgrade", reqUpType)
	}

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := outreq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outreq.Header.Set("X-Forwarded-For", clientIP)
	}

	var apiValidator *OpenAPIValidator
	if p.options.OpenAPI != nil {
		apiValidator = NewOpenAPIValidator(p.options.OpenAPI)
		if roundtripInfo.Err = apiValidator.ValidateRequest(outreq, roundtripInfo); roundtripInfo.Err != nil {
			p.options.ErrorTemplate.ServeError(couperErr.UpstreamRequestValidationFailed).ServeHTTP(rw, req)
			return
		}
	}

	res, err := p.getTransport(outreq.URL.Scheme, outreq.URL.Host, outreq.Host).RoundTrip(outreq)
	roundtripInfo.BeReq, roundtripInfo.BeResp = outreq, res
	if err != nil {
		roundtripInfo.Err = err
		errCode := couperErr.APIConnect
		if strings.HasPrefix(err.Error(), "proxyconnect") {
			errCode = couperErr.APIProxyConnect
		}
		p.options.ErrorTemplate.ServeError(p.getErrorCode(errCode)).ServeHTTP(rw, req)
		return
	}

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if res.StatusCode == http.StatusSwitchingProtocols {
		p.SetRoundtripContext(req, res)
		p.handleUpgradeResponse(rw, outreq, res)
		return
	}

	if strings.ToLower(res.Header.Get(ContentEncodingHeader)) == GzipName {
		var src io.Reader
		var err error

		res.Header.Del(ContentEncodingHeader)

		src, err = gzip.NewReader(res.Body)
		if err != nil {
			src = res.Body
		}

		res.Body = eval.NewReadCloser(src, res.Body)
	}

	removeConnectionHeaders(res.Header)

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	if apiValidator != nil {
		roundtripInfo.Err = apiValidator.ValidateResponse(res, roundtripInfo)
		if roundtripInfo.Err != nil {
			p.options.ErrorTemplate.ServeError(couperErr.UpstreamResponseValidationFailed).ServeHTTP(rw, req)
			return
		}
	}

	p.SetRoundtripContext(req, res)

	copyHeader(rw.Header(), res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	announcedTrailers := len(res.Trailer)
	if announcedTrailers > 0 {
		trailerKeys := make([]string, 0, len(res.Trailer))
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		rw.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	rw.WriteHeader(res.StatusCode)

	_, roundtripInfo.Err = io.Copy(rw, res.Body)

	res.Body.Close() // close now, instead of defer, to populate res.Trailer

	if roundtripInfo.Err != nil {
		return
	}

	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		if fl, ok := rw.(http.Flusher); ok {
			fl.Flush()
		}
	}

	if len(res.Trailer) == announcedTrailers {
		copyHeader(rw.Header(), res.Trailer)
		return
	}

	for k, vv := range res.Trailer {
		k = http.TrailerPrefix + k
		for _, v := range vv {
			rw.Header().Add(k, v)
		}
	}
}

// Director request modification before roundtrip
func (p *Proxy) Director(req *http.Request) error {
	if err := p.SetGetBody(req); err != nil {
		return err
	}

	var origin, hostname, path string
	evalContext := eval.NewHTTPContext(p.evalContext, p.bufferOption, req, nil, nil)

	content, _, _ := p.options.Context.PartialContent(backendInlineSchema)
	if o := getAttribute(evalContext, "origin", content); o != "" {
		origin = o
	}
	if h := getAttribute(evalContext, "hostname", content); h != "" {
		hostname = h
	}
	if pathVal := getAttribute(evalContext, "path", content); pathVal != "" {
		path = pathVal
	}

	if origin == "" {
		return errors.New("proxy: origin not set")
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("proxy: parse origin: %w", err)
	}

	req.URL.Host = originURL.Host
	req.URL.Scheme = originURL.Scheme
	req.Host = originURL.Host

	if hostname != "" {
		req.Host = hostname
	}

	if ReClientSupportsGZ.MatchString(req.Header.Get(AcceptEncodingHeader)) {
		req.Header.Set(AcceptEncodingHeader, GzipName)
	} else {
		req.Header.Del(AcceptEncodingHeader)
	}

	if pathMatch, ok := req.Context().
		Value(request.Wildcard).(string); ok && strings.HasSuffix(path, "/**") {
		if strings.HasSuffix(req.URL.Path, "/") && !strings.HasSuffix(pathMatch, "/") {
			pathMatch += "/"
		}

		req.URL.Path = utils.JoinPath("/", strings.ReplaceAll(path, "/**", "/"), pathMatch)
	} else if path != "" {
		req.URL.Path = utils.JoinPath("/", path)
	}

	p.SetRoundtripContext(req, nil)

	return nil
}

func (p *Proxy) SetRoundtripContext(req *http.Request, beresp *http.Response) {
	var (
		attrCtxAdd = attrAddReqHeaders
		attrCtxDel = attrDelReqHeaders
		attrCtxSet = attrSetReqHeaders
		bereq      *http.Request
		headerCtx  = req.Header
	)

	if beresp != nil {
		attrCtxAdd = attrAddResHeaders
		attrCtxDel = attrDelResHeaders
		attrCtxSet = attrSetResHeaders
		bereq = beresp.Request
		headerCtx = beresp.Header

		defer p.setCorsRespHeaders(headerCtx, req)
	}

	allAttributes, attrOk := p.options.Context.(body.Attributes)
	if !attrOk {
		return
	}

	evalCtx := eval.NewHTTPContext(p.evalContext, p.bufferOption, req, bereq, beresp)

	// Remove blacklisted headers after evaluation to
	// be accessible within our context configuration.
	if attrCtxSet == attrSetReqHeaders {
		for _, key := range headerBlacklist {
			headerCtx.Del(key)
		}
	}

	var modifyQuery bool

	u := *req.URL
	u.RawQuery = strings.ReplaceAll(u.RawQuery, "+", "%2B")
	values := u.Query()

	for _, attrs := range allAttributes.JustAllAttributes() {
		// apply header values in hierarchical and logical order: delete, set, add
		attr, ok := attrs[attrCtxDel]
		if ok {
			val, diags := attr.Expr.Value(evalCtx)
			if seetie.SetSeverityLevel(diags).HasErrors() {
				p.log.WithField("parse config", p.String()).Error(diags)
			}

			for _, key := range seetie.ValueToStringSlice(val) {
				k := http.CanonicalHeaderKey(key)
				if k == "User-Agent" {
					headerCtx[k] = []string{}
					continue
				}

				headerCtx.Del(k)
			}
		}

		attr, ok = attrs[attrCtxSet]
		if ok {
			options, diags := NewOptionsMap(evalCtx, attr)
			if diags != nil {
				p.log.WithField("parse config", p.String()).Error(diags)
			}

			for key, values := range options {
				k := http.CanonicalHeaderKey(key)
				headerCtx[k] = values
			}
		}

		attr, ok = attrs[attrCtxAdd]
		if ok {
			options, diags := NewOptionsMap(evalCtx, attr)
			if diags != nil {
				p.log.WithField("parse config", p.String()).Error(diags)
			}

			for key, values := range options {
				k := http.CanonicalHeaderKey(key)
				headerCtx[k] = append(headerCtx[k], values...)
			}
		}

		if p.options.BasicAuth != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(p.options.BasicAuth))
			headerCtx.Set("Authorization", "Basic "+auth)
		}

		if req == nil || beresp != nil { // just one way -> origin
			continue
		}

		// apply query params in hierarchical and logical order: delete, set, add
		attr, ok = attrs[attrDelQueryParams]
		if ok {
			val, diags := attr.Expr.Value(evalCtx)
			if seetie.SetSeverityLevel(diags).HasErrors() {
				p.log.WithField("parse config", p.String()).Error(diags)
			}

			for _, key := range seetie.ValueToStringSlice(val) {
				values.Del(key)
			}

			modifyQuery = true
		}

		attr, ok = attrs[attrSetQueryParams]
		if ok {
			options, diags := NewOptionsMap(evalCtx, attr)
			if diags != nil {
				p.log.WithField("parse config", p.String()).Error(diags)
			}

			for k, v := range options {
				values[k] = v
			}

			modifyQuery = true
		}

		attr, ok = attrs[attrAddQueryParams]
		if ok {
			options, diags := NewOptionsMap(evalCtx, attr)
			if diags != nil {
				p.log.WithField("parse config", p.String()).Error(diags)
			}

			for k, v := range options {
				if _, ok = values[k]; !ok {
					values[k] = v
				} else {
					values[k] = append(values[k], v...)
				}
			}

			modifyQuery = true
		}
	}

	if modifyQuery {
		req.URL.RawQuery = strings.ReplaceAll(values.Encode(), "+", "%20")
	}
}

// SetGetBody determines if we have to buffer a request body for further processing.
// First of all the user has a related reference within a config.Backend options declaration.
// Additionally the request body is nil or a NoBody type and the http method has no body restrictions like 'TRACE'.
func (p *Proxy) SetGetBody(req *http.Request) error {
	if req.Method == http.MethodTrace {
		return nil
	}

	if (p.bufferOption & eval.BufferRequest) != eval.BufferRequest {
		return nil
	}

	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		buf := &bytes.Buffer{}
		lr := io.LimitReader(req.Body, p.options.RequestBodyLimit+1)
		n, err := buf.ReadFrom(lr)
		if err != nil {
			return err
		}

		if n > p.options.RequestBodyLimit {
			return p.getErrorCode(couperErr.APIReqBodySizeExceeded)
		}

		bodyBytes := buf.Bytes()
		req.GetBody = func() (io.ReadCloser, error) {
			return eval.NewReadCloser(bytes.NewBuffer(bodyBytes), req.Body), nil
		}
	}

	return nil
}

func isCorsRequest(req *http.Request) bool {
	return req.Header.Get("Origin") != ""
}

func isCorsPreflightRequest(req *http.Request) bool {
	return req.Method == http.MethodOptions && (req.Header.Get("Access-Control-Request-Method") != "" || req.Header.Get("Access-Control-Request-Headers") != "")
}

func IsCredentialed(headers http.Header) bool {
	return headers.Get("Cookie") != "" || headers.Get("Authorization") != "" || headers.Get("Proxy-Authorization") != ""
}

func (p *Proxy) setCorsRespHeaders(headers http.Header, req *http.Request) {
	if p.options.CORS == nil || !isCorsRequest(req) {
		return
	}
	requestOrigin := req.Header.Get("Origin")
	if !p.options.CORS.AllowsOrigin(requestOrigin) {
		return
	}
	// see https://fetch.spec.whatwg.org/#http-responses
	if p.options.CORS.AllowsOrigin("*") && !IsCredentialed(req.Header) {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", requestOrigin)
	}

	if p.options.CORS.AllowCredentials == true {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	if isCorsPreflightRequest(req) {
		// Reflect request header value
		acrm := req.Header.Get("Access-Control-Request-Method")
		if acrm != "" {
			headers.Set("Access-Control-Allow-Methods", acrm)
		}
		// Reflect request header value
		acrh := req.Header.Get("Access-Control-Request-Headers")
		if acrh != "" {
			headers.Set("Access-Control-Allow-Headers", acrh)
		}
		if p.options.CORS.MaxAge != "" {
			headers.Set("Access-Control-Max-Age", p.options.CORS.MaxAge)
		}
	} else if p.options.CORS.NeedsVary() {
		headers.Add("Vary", "Origin")
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

func upgradeType(h http.Header) string {
	if !httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade") {
		return ""
	}
	return strings.ToLower(h.Get("Upgrade"))
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (p *Proxy) handleUpgradeResponse(rw http.ResponseWriter, req *http.Request, res *http.Response) {
	reqUpType := upgradeType(req.Header)
	resUpType := upgradeType(res.Header)
	if reqUpType != resUpType {
		p.log.Errorf("backend tried to switch protocol %q when %q was requested", resUpType, reqUpType)
		return
	}

	copyHeader(res.Header, rw.Header())

	hj, ok := rw.(http.Hijacker)
	if !ok {
		p.log.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", rw)
		return
	}
	backConn, ok := res.Body.(io.ReadWriteCloser)
	if !ok {
		p.log.Errorf("internal error: 101 switching protocols response with non-writable body")
		return
	}
	defer backConn.Close()
	conn, brw, err := hj.Hijack()
	if err != nil {
		p.log.Errorf("hijack failed on protocol switch: %v", err)
		return
	}
	defer conn.Close()
	res.Body = nil // so res.Write only writes the headers; we have res.Body in backConn above
	if err = res.Write(brw); err != nil {
		p.log.Errorf("response write: %v", err)
		return
	}
	if err = brw.Flush(); err != nil {
		p.log.Errorf("response flush: %v", err)
		return
	}
	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: conn, backend: backConn}
	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)
	<-errc
	return
}

func (p *Proxy) Options() *server.Options {
	return p.srvOptions
}

func (p *Proxy) String() string {
	return p.options.Kind
}

func (p *Proxy) getErrorCode(code couperErr.Code) couperErr.Code {
	if p.options.Kind == "endpoint" {
		switch code {
		case couperErr.APIConnect:
			return couperErr.EndpointConnect
		case couperErr.APIProxyConnect:
			return couperErr.EndpointProxyConnect
		case couperErr.APIReqBodySizeExceeded:
			return couperErr.EndpointReqBodySizeExceeded
		}

		return couperErr.EndpointError
	}

	return code
}

func getAttribute(ctx *hcl.EvalContext, name string, body *hcl.BodyContent) string {
	attr := body.Attributes
	if _, ok := attr[name]; !ok {
		return ""
	}
	originValue, _ := attr[name].Expr.Value(ctx)
	return seetie.ValueToString(originValue)
}

// switchProtocolCopier exists so goroutines proxying data back and
// forth have nice names in stacks.
type switchProtocolCopier struct {
	user, backend io.ReadWriter
}

func (c switchProtocolCopier) copyFromBackend(errc chan<- error) {
	_, err := io.Copy(c.user, c.backend)
	errc <- err
}

func (c switchProtocolCopier) copyToBackend(errc chan<- error) {
	_, err := io.Copy(c.backend, c.user)
	errc <- err
}
