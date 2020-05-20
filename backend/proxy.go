package backend

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	_ http.Handler = &Proxy{}
)

type Proxy struct {
	OriginAddress string
	OriginHost    string
	rp            *httputil.ReverseProxy
	log           *logrus.Entry
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.rp.ServeHTTP(rw, req)
}

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.OriginAddress
	req.URL.Scheme = "http"
	if strings.HasSuffix(p.OriginAddress, "443") {
		req.URL.Scheme = "https" // TODO: improve conf options, scheme or url
	}
	req.Host = p.OriginHost
}

func (p *Proxy) Init() { // TODO: some kind of factory -> config.Load
	p.rp = &httputil.ReverseProxy{Director: p.director}
}

func (p *Proxy) String() string {
	return "Proxy"
}
