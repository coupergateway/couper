package validation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval/buffer"
)

type OpenAPIOptions struct {
	buffer                   buffer.Option
	ignoreRequestViolations  bool
	ignoreResponseViolations bool
	filterOptions            *openapi3filter.Options
	doc                      *openapi3.T
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

	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return NewOpenAPIOptionsFromBytes(openapi, b)
}

func NewOpenAPIOptionsFromBytes(openapi *config.OpenAPI, bytes []byte) (*OpenAPIOptions, error) {
	if openapi == nil || bytes == nil {
		return nil, nil
	}

	doc, err := openapi3.NewLoader().LoadFromData(bytes)
	if err != nil {
		return nil, fmt.Errorf("error loading openapi file: %w", err)
	}

	// Always buffer if openAPI is active. Request buffering is handled by openapifilter too.
	// Anyway adding request buffer option to let Couper check the body limits.
	bufferBodies := buffer.Request | buffer.Response

	return &OpenAPIOptions{
		buffer: bufferBodies,
		filterOptions: &openapi3filter.Options{
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
		},
		ignoreRequestViolations:  openapi.IgnoreRequestViolations,
		ignoreResponseViolations: openapi.IgnoreResponseViolations,
		doc:                      doc,
	}, nil
}
