package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"time"
)

type Backend struct {
	srv *httptest.Server
	mux *http.ServeMux
}

func NewBackend() *Backend {
	b := &Backend{
		mux: http.NewServeMux(),
	}

	b.srv = httptest.NewServer(b)

	// test handler
	b.mux.HandleFunc("/anything", anything)

	return b
}

func (b *Backend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.mux.ServeHTTP(rw, req)
}

func (b *Backend) Close() {
	b.srv.Close()
}

func (b *Backend) Addr() string {
	return b.srv.URL
}

func anything(rw http.ResponseWriter, req *http.Request) {
	type anything struct {
		Args                               url.Values
		Headers                            http.Header
		Host                               string
		Path                               string
		Method, RemoteAddr, Url, UserAgent string
	}

	_ = req.ParseForm()

	resp := &anything{
		Args:       req.Form,
		Headers:    req.Header.Clone(),
		Host:       req.Host,
		Method:     req.Method,
		Path:       req.URL.Path,
		RemoteAddr: req.RemoteAddr,
		Url:        req.URL.String(),
		UserAgent:  req.UserAgent(),
	}

	respContent, _ := json.Marshal(resp)

	rw.Header().Set("Server", "couper test-backend")
	rw.Header().Set("Date", time.Now().Format(http.TimeFormat))
	rw.Header().Set("Content-Length", strconv.Itoa(len(respContent)))
	rw.Header().Set("Content-Type", "application/json")

	_, _ = rw.Write(respContent)
}
