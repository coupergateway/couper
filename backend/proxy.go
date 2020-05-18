package backend

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

var _ http.Handler = &Proxy{}

type Proxy struct {
	log *logrus.Entry
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

}
