package transport

import (
	"net/http"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/handler/validation"
)

// BackendOptions represents the transport <BackendOptions> object.
type BackendOptions struct {
	RequestAuthz []RequestAuthorizer
	HealthCheck  *config.HealthCheck
	OpenAPI      *validation.OpenAPIOptions
}

type RequestAuthorizer interface {
	GetToken(req *http.Request) error
	RetryWithToken(req *http.Request, res *http.Response) (bool, error)

	value() (string, string)
}
