package validation

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
)

type OpenAPIOptions struct {
	buffer                   eval.BufferOption
	ignoreRequestViolations  bool
	ignoreResponseViolations bool
	filterOptions            *openapi3filter.Options
	swagger                  *openapi3.Swagger
}

// NewOpenAPIOptions takes a list of openAPI configuration due to merging configurations.
// The last item will be set and no attributes gets merged.
func NewOpenAPIOptions(openapi *config.OpenAPI) (*OpenAPIOptions, error) {
	if openapi == nil {
		return nil, nil
	}

	p, err := filepath.Abs(openapi.File)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return NewOpenAPIOptionsFromBytes(openapi, b)
}

func canonicalizeServerURLs(swagger *openapi3.Swagger) error {
	for _, server := range swagger.Servers {
		su, err := url.Parse(server.URL)
		if err != nil {
			return err
		}

		if su.IsAbs() && su.Port() == "" && (su.Scheme == "https" || su.Scheme == "http") {
			su.Host = su.Hostname() + ":"
			if su.Scheme == "https" {
				su.Host += "443"
			} else {
				su.Host += "80"
			}
			server.URL = su.String()
		}
	}
	return nil
}

func NewOpenAPIOptionsFromBytes(openapi *config.OpenAPI, bytes []byte) (*OpenAPIOptions, error) {
	if openapi == nil || bytes == nil {
		return nil, nil
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(bytes)
	if err != nil {
		return nil, fmt.Errorf("error loading openapi file: %w", err)
	}

	if err = canonicalizeServerURLs(swagger); err != nil {
		return nil, err
	}

	// Always buffer if openAPI is active. Request buffering is handled by openapifilter too.
	// Anyway adding request buffer option to let Couper check the body limits.
	bufferBodies := eval.BufferRequest | eval.BufferResponse

	return &OpenAPIOptions{
		buffer: bufferBodies,
		filterOptions: &openapi3filter.Options{
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
		},
		ignoreRequestViolations:  openapi.IgnoreRequestViolations,
		ignoreResponseViolations: openapi.IgnoreResponseViolations,
		swagger:                  swagger,
	}, nil
}
