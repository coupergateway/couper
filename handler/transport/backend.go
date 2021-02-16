package transport

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

var _ http.RoundTripper = &Backend{}

// Backend represents the transport <Backend> object.
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

// NewBackend creates a new <*Backend> object by the given <*Config>.
func NewBackend(conf *Config) *Backend {
	return &Backend{transportConf: conf}
}

// RoundTrip implements the <http.RoundTripper> interface.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	return Get(b.transportConf).RoundTrip(req)
}
