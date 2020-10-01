package handler

import (
	"net/http"
	"strings"

	"github.com/avenga/couper/errors"
)

const healthPath = "/healthz"

var _ http.Handler = &Health{}

type Health struct {
	path       string
	shutdownCh chan struct{}
}

func NewHealthCheck(path string, shutdownCh chan struct{}) *Health {
	p := path
	if p == "" {
		p = healthPath
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	return &Health{
		path:       p,
		shutdownCh: shutdownCh,
	}
}

func (h *Health) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Content-Type", "text/plain")

	select {
	case <-h.shutdownCh:
		errors.SetHeader(rw, errors.ServerShutdown)
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("server shutting down"))
	default:
		_, _ = rw.Write([]byte("healthy"))
	}
}

func (h *Health) Match(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, h.path)
}

func (h *Health) String() string {
	return "health"
}