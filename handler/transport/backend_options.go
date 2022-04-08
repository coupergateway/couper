package transport

import (
	"net/http"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/handler/validation"
)

// BackendOptions represents the transport <BackendOptions> object.
type BackendOptions struct {
	AuthBackend TokenRequest
	HealthCheck *config.HealthCheck
	OpenAPI     *validation.OpenAPIOptions
}

type TokenRequest interface {
	WithToken(req *http.Request) error
	RetryWithToken(req *http.Request, res *http.Response) (bool, error)
}
