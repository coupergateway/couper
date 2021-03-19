package transport

import "github.com/avenga/couper/handler/validation"

// BackendOptions represents the transport <BackendOptions> object.
type BackendOptions struct {
	OpenAPI *validation.OpenAPIOptions
}
