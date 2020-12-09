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
	b.mux.HandleFunc("/anything", createAnythingHandler(http.StatusOK))
	b.mux.HandleFunc("/", createAnythingHandler(http.StatusNotFound))

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

func createAnythingHandler(status int) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		type anything struct {
			Args, Query                        url.Values
			Headers                            http.Header
			Host                               string
			Path                               string
			Method, RemoteAddr, Url, UserAgent string
			ResponseStatus                     int
		}

		_ = req.ParseForm()

		resp := &anything{
			Args:           req.Form,
			Headers:        req.Header.Clone(),
			Host:           req.Host,
			Method:         req.Method,
			Path:           req.URL.Path,
			RemoteAddr:     req.RemoteAddr,
			Query:          req.URL.Query(),
			Url:            req.URL.String(),
			UserAgent:      req.UserAgent(),
			ResponseStatus: status,
		}

		respContent, _ := json.Marshal(resp)

		rw.Header().Set("Server", "couper test-backend")
		rw.Header().Set("Date", time.Now().Format(http.TimeFormat))
		rw.Header().Set("Content-Length", strconv.Itoa(len(respContent)))
		rw.Header().Set("Content-Type", "application/json")

		rw.WriteHeader(status)
		_, _ = rw.Write(respContent)
	}
}
