package handler

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/logging"
)

type OpenAPIValidatorOptions struct {
	buffer                   eval.BufferOption
	ignoreRequestViolations  bool
	ignoreResponseViolations bool
	filterOptions            *openapi3filter.Options
	router                   *openapi3filter.Router
}

// NewOpenAPIValidatorOptions takes a list of openAPI configuration due to merging configurations.
// The last item will be set and no attributes gets merged.
func NewOpenAPIValidatorOptions(openapi []*config.OpenAPI) (*OpenAPIValidatorOptions, error) {
	if len(openapi) == 0 {
		return nil, nil
	}

	openapiBlock := openapi[len(openapi)-1]

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(filepath.Join(dir, openapiBlock.File))
	if err != nil {
		return nil, err
	}
	return NewOpenAPIValidatorOptionsFromBytes(openapiBlock, b)
}

func NewOpenAPIValidatorOptionsFromBytes(openapi *config.OpenAPI, bytes []byte) (*OpenAPIValidatorOptions, error) {
	if openapi == nil || bytes == nil {
		return nil, nil
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(bytes)
	if err != nil {
		return nil, fmt.Errorf("error loading openapi file: %w", err)
	}

	router := openapi3filter.NewRouter()
	if err = router.AddSwagger(swagger); err != nil {
		return nil, err
	}

	// Always buffer if openAPI is active. Request buffering is handled by openapifilter too.
	// Anyway adding request buffer option to let Couper check the body limits.
	bufferBodies := eval.BufferRequest | eval.BufferResponse

	return &OpenAPIValidatorOptions{
		buffer: bufferBodies,
		filterOptions: &openapi3filter.Options{
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
		},
		ignoreRequestViolations:  openapi.IgnoreRequestViolations,
		ignoreResponseViolations: openapi.IgnoreResponseViolations,
		router:                   router,
	}, nil
}

func NewOpenAPIValidator(opts *OpenAPIValidatorOptions) *OpenAPIValidator {
	return &OpenAPIValidator{
		options: opts,
	}
}

type OpenAPIValidator struct {
	options                *OpenAPIValidatorOptions
	requestValidationInput *openapi3filter.RequestValidationInput
}

func (v *OpenAPIValidator) ValidateRequest(req *http.Request, tripInfo *logging.RoundtripInfo) error {
	route, pathParams, err := v.options.router.FindRoute(req.Method, req.URL)
	if err != nil {
		err = fmt.Errorf("request validation: '%s %s': %w", req.Method, req.URL.Path, err)
		if !v.options.ignoreRequestViolations {
			return err
		}
		tripInfo.ValidationError = append(tripInfo.ValidationError, err)
		return nil
	}

	v.requestValidationInput = &openapi3filter.RequestValidationInput{
		Options:     v.options.filterOptions,
		PathParams:  pathParams,
		QueryParams: req.URL.Query(),
		Request:     req,
		Route:       route,
	}

	// openapi3filter.ValidateRequestBody also handles resetting the req body after reading until EOF.
	err = openapi3filter.ValidateRequest(req.Context(), v.requestValidationInput)

	if err != nil {
		err = fmt.Errorf("request validation: %w", err)
		if !v.options.ignoreRequestViolations {
			return err
		}
		tripInfo.ValidationError = append(tripInfo.ValidationError, err)
	}

	return nil
}

func (v *OpenAPIValidator) ValidateResponse(beresp *http.Response, tripInfo *logging.RoundtripInfo) error {
	// since a request validation could fail and ignored due to user options, the input route MAY be nil
	if v.requestValidationInput == nil || v.requestValidationInput.Route == nil {
		err := fmt.Errorf("response validation: '%s %s': invalid route", beresp.Request.Method, beresp.Request.URL.Path)
		if v.options.ignoreResponseViolations {
			tripInfo.ValidationError = append(tripInfo.ValidationError, err)
			return nil
		}
		// Since a matching route is required; we are done here.
		return err
	}

	responseValidationInput := &openapi3filter.ResponseValidationInput{
		Body:                   ioutil.NopCloser(&bytes.Buffer{}),
		Header:                 beresp.Header.Clone(),
		Options:                v.options.filterOptions,
		RequestValidationInput: v.requestValidationInput,
		Status:                 beresp.StatusCode,
	}

	if !v.options.filterOptions.ExcludeResponseBody {
		// buffer beresp body
		buf := &bytes.Buffer{}
		_, err := io.Copy(buf, beresp.Body)
		if err != nil {
			return err
		}
		// reset and provide the buffer
		beresp.Body = eval.NewReadCloser(buf, beresp.Body)
		// provide a copy for validation purposes
		responseValidationInput.SetBodyBytes(buf.Bytes())
	}

	if err := openapi3filter.ValidateResponse(beresp.Request.Context(), responseValidationInput); err != nil {
		err = fmt.Errorf("response validation: %w", err)
		if !v.options.ignoreResponseViolations {
			return err
		}
		tripInfo.ValidationError = append(tripInfo.ValidationError, err)
	}

	return nil
}
