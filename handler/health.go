package handler

import (
	"net/http"
	"strings"
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
	return &Health{
		path:       p,
		shutdownCh: shutdownCh,
	}
}

func (h *Health) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	select {
	case <-h.shutdownCh:
		rw.WriteHeader(http.StatusInternalServerError)
	default:
		rw.WriteHeader(http.StatusOK)
	}
}

func (h *Health) Match(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, h.path)
}
