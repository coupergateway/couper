package transport

import (
	"net/http"

	"github.com/avenga/couper/handler/validation"
)

// BackendOptions represents the transport <BackendOptions> object.
type BackendOptions struct {
	OpenAPI     *validation.OpenAPIOptions
	AuthBackend TokenRequest
}

type TokenRequest interface {
	WithToken(req *http.Request) error
	RetryWithToken(req *http.Request, res *http.Response) (bool, error)
}
