package backend

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

var _ http.Handler = &Proxy{}

type Proxy struct {
	OriginAddress string
	OriginHost    string
	log           *logrus.Entry
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	println("proxy roxy")
}

func (p *Proxy) String() string {
	return "Proxy"
}
