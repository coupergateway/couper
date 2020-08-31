package handler

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpguts"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/eval"
)

var (
	_ http.Handler = &Proxy{}

	OriginRequiredError = errors.New("origin is required")
	SchemeRequiredError = errors.New("backend origin must define a scheme")

	// headerBlacklist lists all header keys which will be removed after
	// context variable evaluation to ensure to not pass them upstream.
	headerBlacklist = []string{"Authorization", "Cookie"}
)

type Proxy struct {
	evalContext *hcl.EvalContext
	log         *logrus.Entry
	options     *ProxyOptions
	originURL   *url.URL
	transport   *http.Transport
}

type ProxyOptions struct {
	ConnectTimeout, Timeout, TTFBTimeout time.Duration
	Context                              []hcl.Body
	Hostname, Origin, Path               string
}

func NewProxy(options *ProxyOptions, log *logrus.Entry, evalCtx *hcl.EvalContext) (http.Handler, error) {
	if options.Origin == "" {
		return nil, OriginRequiredError
	}
	originURL, err := url.Parse(options.Origin)
	if err != nil {
		return nil, fmt.Errorf("err parsing origin url: %w", err)
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return nil, SchemeRequiredError
	}

	proxy := &Proxy{
		evalContext: evalCtx,
		log:         log,
		options:     options,
		originURL:   originURL,
	}

	var tlsConf *tls.Config
	if options.Hostname != "" {
		tlsConf = &tls.Config{
			ServerName: options.Hostname,
		}
	}

	d := &net.Dialer{Timeout: options.ConnectTimeout}
	proxy.transport = &http.Transport{
		// DisableCompression: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := d.DialContext(ctx, network, addr)
			if err != nil {
				return nil, fmt.Errorf("connecting to %s failed: %w", originURL.String(), err)
			}
			return conn, nil
		},
		ResponseHeaderTimeout: proxy.options.TTFBTimeout,
		TLSClientConfig:       tlsConf,
	}
	return proxy, nil
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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

	p.director(outreq)
	outreq.Close = false

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

	res, err := p.transport.RoundTrip(outreq)
	if err != nil {
		p.log.
			WithField("type", "access").Errorf("request error: %s: %v", outreq.URL.String(), err)
		return
	}

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if res.StatusCode == http.StatusSwitchingProtocols {
		p.setRoundtripContext(req, res)
		p.handleUpgradeResponse(rw, outreq, res)
		return
	}

	removeConnectionHeaders(res.Header)

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	p.setRoundtripContext(req, res)

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

	_, err = io.Copy(rw, res.Body)
	if err != nil {
		defer res.Body.Close()
		p.log.Error(err)
		return
	}

	res.Body.Close() // close now, instead of defer, to populate res.Trailer

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

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.originURL.Host
	req.URL.Scheme = p.originURL.Scheme
	req.Host = p.originURL.Host
	if p.options.Hostname != "" {
		req.Host = p.options.Hostname
	}

	if pathMatch, ok := req.Context().
		Value(config.WildcardCtxKey).(string); ok && strings.HasSuffix(p.options.Path, "/**") {
		req.URL.Path = path.Join(strings.ReplaceAll(p.options.Path, "/**", "/"), pathMatch)
	} else if p.options.Path != "" {
		req.URL.Path = p.options.Path
	}

	p.setRoundtripContext(req, nil)
}

func (p *Proxy) setRoundtripContext(req *http.Request, beresp *http.Response) {
	var (
		attrCtx   = attrReqHeaders
		bereq     *http.Request
		headerCtx http.Header
	)

	if beresp != nil {
		attrCtx = attrResHeaders
		bereq = beresp.Request
		headerCtx = beresp.Header
	} else if req != nil {
		headerCtx = req.Header
	}

	evalCtx := eval.NewHTTPContext(p.evalContext, req, bereq, beresp)

	// Remove blacklisted headers after evaluation to be accessible within our context configuration.
	if attrCtx == attrReqHeaders {
		for _, key := range headerBlacklist {
			headerCtx.Del(key)
		}
	}

	for _, ctxBody := range p.options.Context {
		options, err := NewCtxOptions(attrCtx, evalCtx, ctxBody)
		if err != nil {
			p.log.WithField("type", "couper_hcl").WithField("parse config", p.String()).Error(err)
		}
		setHeaderFields(headerCtx, options)
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
		p.log.Error(fmt.Errorf("backend tried to switch protocol %q when %q was requested", resUpType, reqUpType))
		return
	}

	copyHeader(res.Header, rw.Header())

	hj, ok := rw.(http.Hijacker)
	if !ok {
		p.log.Error(fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", rw))
		return
	}
	backConn, ok := res.Body.(io.ReadWriteCloser)
	if !ok {
		p.log.Error(fmt.Errorf("internal error: 101 switching protocols response with non-writable body"))
		return
	}
	defer backConn.Close()
	conn, brw, err := hj.Hijack()
	if err != nil {
		p.log.Error(fmt.Errorf("hijack failed on protocol switch: %v", err))
		return
	}
	defer conn.Close()
	res.Body = nil // so res.Write only writes the headers; we have res.Body in backConn above
	if err := res.Write(brw); err != nil {
		p.log.Error(fmt.Errorf("response write: %v", err))
		return
	}
	if err := brw.Flush(); err != nil {
		p.log.Error(fmt.Errorf("response flush: %v", err))
		return
	}
	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: conn, backend: backConn}
	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)
	<-errc
	return
}

func (p *Proxy) String() string {
	return "Proxy"
}

func setHeaderFields(header http.Header, options OptionsMap) {
	if len(options) == 0 {
		return
	}

	for key, value := range options {
		if len(value) == 0 || value[0] == "" {
			header.Del(key)
			continue
		}
		k := http.CanonicalHeaderKey(key)
		header[k] = value
	}
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
