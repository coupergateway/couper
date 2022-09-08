package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"

	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/ascii"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/server/writer"
)

// headerBlacklist lists all header keys which will be removed after
// context variable evaluation to ensure to not pass them upstream.
var headerBlacklist = []string{"Authorization", "Cookie"}

// Proxy wraps a httputil.ReverseProxy to apply additional configuration context
// and have control over the roundtrip configuration.
type Proxy struct {
	backend http.RoundTripper
	context *hclsyntax.Body
	logger  *logrus.Entry
}

func NewProxy(backend http.RoundTripper, ctx *hclsyntax.Body, logger *logrus.Entry) *Proxy {
	proxy := &Proxy{
		backend: backend,
		context: ctx,
		logger:  logger,
	}

	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Apply proxy blacklist
	for _, key := range headerBlacklist {
		req.Header.Del(key)
	}

	hclCtx := eval.ContextFromRequest(req).HCLContextSync()

	// 2. Apply proxy-body
	if err := eval.ApplyRequestContext(hclCtx, p.context, req); err != nil {
		return nil, err
	}

	// 3. Apply websockets-body
	if err := p.applyWebsocketsRequest(req); err != nil {
		return nil, err
	}

	// 4. apply some hcl context
	expStatusVal, err := eval.ValueFromBodyAttribute(hclCtx, p.context, "expected_status")
	if err != nil {
		return nil, err
	}

	outCtx := context.WithValue(req.Context(), request.EndpointExpectedStatus, seetie.ValueToIntSlice(expStatusVal))

	*req = *req.WithContext(outCtx)

	if err = p.registerWebsocketsResponse(req); err != nil {
		return nil, err
	}

	// the chore reverse-proxy part
	if req.ContentLength == 0 {
		req.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	if req.Body != nil {
		defer req.Body.Close()
	}
	req.Close = false

	reqUpType := upgradeType(req.Header)
	if !ascii.IsPrint(reqUpType) {
		return nil, fmt.Errorf("client tried to switch to invalid protocol %q", reqUpType)
	}

	transport.RemoveConnectionHeaders(req.Header)

	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range transport.HopHeaders {
		req.Header.Del(h)
	}

	// TODO: trailer header here

	// After stripping all the hop-by-hop connection headers above, add back any
	// necessary for protocol upgrades, such as for websockets.
	if reqUpType != "" {
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", reqUpType)
	}

	beresp, err := p.backend.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if beresp.StatusCode == http.StatusSwitchingProtocols {
		return beresp, p.handleUpgradeResponse(req, beresp)
	}

	transport.RemoveConnectionHeaders(beresp.Header)
	transport.RemoveHopHeaders(beresp.Header)

	evalCtx := eval.ContextFromRequest(req)
	err = eval.ApplyResponseContext(evalCtx.HCLContextSync(), p.context, beresp)

	return beresp, err
}

func upgradeType(h http.Header) string {
	conn, exist := h["Connection"]
	if !exist {
		return ""
	}
	for _, v := range conn {
		if strings.ToLower(v) == "upgrade" {
			return h.Get("Upgrade")
		}
	}
	return ""
}

func (p *Proxy) applyWebsocketsRequest(req *http.Request) error {
	ctx := req.Context()

	ctx = context.WithValue(ctx, request.WebsocketsAllowed, true)
	*req = *req.WithContext(ctx)

	hclCtx := eval.ContextFromRequest(req).HCLContextSync()

	// This method needs the 'request.WebsocketsAllowed' flag in the 'req.context'.
	if !eval.IsUpgradeRequest(req) {
		return nil
	}

	wsBody := p.getWebsocketsBody()
	if err := eval.ApplyRequestContext(hclCtx, wsBody, req); err != nil {
		return err
	}

	attr, ok := wsBody.Attributes["timeout"]
	if !ok {
		return nil
	}

	val, err := eval.Value(hclCtx, attr.Expr)
	if err != nil {
		return err
	}

	str := seetie.ValueToString(val)

	timeout, err := time.ParseDuration(str)
	if str != "" && err != nil {
		return err
	}

	ctx = context.WithValue(ctx, request.WebsocketsTimeout, timeout)
	*req = *req.WithContext(ctx)

	return nil
}

func (p *Proxy) registerWebsocketsResponse(req *http.Request) error {
	ctx := req.Context()

	ctx = context.WithValue(ctx, request.WebsocketsAllowed, true)
	*req = *req.WithContext(ctx)

	// This method needs the 'request.WebsocketsAllowed' flag in the 'req.context'.
	if !eval.IsUpgradeRequest(req) {
		return nil
	}

	wsBody := p.getWebsocketsBody()
	evalCtx := eval.ContextFromRequest(req)

	if rw, ok := req.Context().Value(request.ResponseWriter).(*writer.Response); ok {
		rw.AddModifier(evalCtx.HCLContextSync(), wsBody, p.context)
	}

	return nil
}

func (p *Proxy) getWebsocketsBody() *hclsyntax.Body {
	wss := hclbody.BlocksOfType(p.context, "websockets")
	if len(wss) != 1 {
		return nil
	}

	return wss[0].Body
}

func (p *Proxy) handleUpgradeResponse(req *http.Request, res *http.Response) error {
	rw, ok := req.Context().Value(request.ResponseWriter).(http.ResponseWriter)
	if !ok {
		return fmt.Errorf("can't switch protocols using non-ResponseWriter type %T", rw)
	}

	reqUpType := upgradeType(req.Header)
	resUpType := upgradeType(res.Header)
	if !ascii.IsPrint(resUpType) { // We know reqUpType is ASCII, it's checked by the caller.
		return fmt.Errorf("backend tried to switch to invalid protocol %q", resUpType)
	}
	if !ascii.EqualFold(reqUpType, resUpType) {
		return fmt.Errorf("backend tried to switch protocol %q when %q was requested", resUpType, reqUpType)
	}

	hj, ok := rw.(http.Hijacker)
	if !ok {
		return fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", rw)
	}
	backConn, ok := res.Body.(io.ReadWriteCloser)
	if !ok {
		return fmt.Errorf("internal error: 101 switching protocols response with non-writable body")
	}

	backConnCloseCh := make(chan bool)
	go func() {
		// Ensure that the cancellation of a request closes the backend.
		// See issue https://golang.org/issue/35559.
		select {
		case <-req.Context().Done():
		case <-backConnCloseCh:
		}
		backConn.Close()
	}()

	defer close(backConnCloseCh)

	conn, brw, err := hj.Hijack()
	if err != nil {
		return fmt.Errorf("hijack failed on protocol switch: %v", err)
	}
	defer conn.Close()

	copyHeader(rw.Header(), res.Header)

	res.Header = rw.Header()
	res.Body = nil // so res.Write only writes the headers; we have res.Body in backConn above
	if err := res.Write(brw); err != nil {
		return fmt.Errorf("response write: %v", err)
	}
	if err := brw.Flush(); err != nil {
		return fmt.Errorf("response flush: %v", err)
	}
	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: conn, backend: backConn}
	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)
	<-errc
	return nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func flushInterval(res *http.Response) time.Duration {
	resCT := res.Header.Get("Content-Type")

	// For Server-Sent Events responses, flush immediately.
	// The MIME type is defined in https://www.w3.org/TR/eventsource/#text-event-stream
	if resCT == "text/event-stream" {
		return -1 // negative means immediately
	}

	// We might have the case of streaming for which Content-Length might be unset.
	if res.ContentLength == -1 {
		return -1
	}

	return time.Millisecond * 100
}

var bufferPool httputil.BufferPool

func copyResponse(dst io.Writer, src io.Reader, flushInterval time.Duration) error {
	if flushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: flushInterval,
			}
			defer mlw.stop()

			// set up initial timer so headers get flushed even if body writes are delayed
			mlw.flushPending = true
			mlw.t = time.AfterFunc(flushInterval, mlw.delayedFlush)

			dst = mlw
		}
	}

	var buf []byte
	if bufferPool != nil {
		buf = bufferPool.Get()
		defer bufferPool.Put(buf)
	}
	_, err := copyBuffer(dst, src, buf)
	return err
}

// copyBuffer returns any write errors or non-EOF read errors, and the amount
// of bytes written.
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			return 0, errors.Server.With(rerr).Message("read error during body copy")
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}

type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration // non-zero; negative means to flush immediately

	mu           sync.Mutex // protects t, flushPending, and dst.Flush
	t            *time.Timer
	flushPending bool
}

func (m *maxLatencyWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, err = m.dst.Write(p)
	if m.latency < 0 {
		m.dst.Flush()
		return
	}
	if m.flushPending {
		return
	}
	if m.t == nil {
		m.t = time.AfterFunc(m.latency, m.delayedFlush)
	} else {
		m.t.Reset(m.latency)
	}
	m.flushPending = true
	return
}

func (m *maxLatencyWriter) delayedFlush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.flushPending { // if stop was called but AfterFunc already started this goroutine
		return
	}
	m.dst.Flush()
	m.flushPending = false
}

func (m *maxLatencyWriter) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushPending = false
	if m.t != nil {
		m.t.Stop()
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
