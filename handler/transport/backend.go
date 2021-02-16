package transport

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

var _ http.RoundTripper = &Backend{}

type Backend struct {
	context       hcl.Body
	name          string
	transportConf *Config
	AccessControl string // maps to basic-auth atm
	//OpenAPI       *OpenAPIValidatorOptions
	// oauth
	// ...
	// TODO: OrderedList for origin AC, middlewares etc.
}

func NewBackend(conf *Config) *Backend {
	return &Backend{transportConf: conf}
}

func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	return Get(b.transportConf).RoundTrip(req)
}
