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

func NewOpenAPIValidatorOptions(openapi *config.OpenAPI) (*OpenAPIValidatorOptions, error) {
	if openapi == nil {
		return nil, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(filepath.Join(dir, openapi.File))
	if err != nil {
		return nil, err
	}
	return NewOpenAPIValidatorOptionsFromBytes(openapi, b)
}

func NewOpenAPIValidatorOptionsFromBytes(openapi *config.OpenAPI, bytes []byte) (*OpenAPIValidatorOptions, error) {
	if openapi == nil || bytes == nil {
		return nil, nil
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(bytes)
	if err != nil {
		return nil, err
	}

	router := openapi3filter.NewRouter()
	if err = router.AddSwagger(swagger); err != nil {
		return nil, err
	}

	bufferBodies := eval.BufferRequest | eval.BufferResponse
	if openapi.ExcludeRequestBody {
		bufferBodies ^= eval.BufferRequest
	}
	if openapi.ExcludeResponseBody {
		bufferBodies ^= eval.BufferResponse
	}

	return &OpenAPIValidatorOptions{
		buffer: bufferBodies,
		filterOptions: &openapi3filter.Options{
			ExcludeRequestBody:    openapi.ExcludeRequestBody,
			ExcludeResponseBody:   openapi.ExcludeResponseBody,
			IncludeResponseStatus: !openapi.ExcludeStatusCode,
		},
		ignoreRequestViolations:  openapi.IgnoreRequestViolations,
		ignoreResponseViolations: openapi.IgnoreResponseViolations,
		router:                   router,
	}, nil
}

func NewOpenAPIValidator(opts *OpenAPIValidatorOptions) *OpenAPIValidator {
	return &OpenAPIValidator{
		options:       opts,
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

	err = openapi3filter.ValidateRequest(req.Context(), v.requestValidationInput)

	if !v.options.filterOptions.ExcludeRequestBody && req.GetBody != nil {
		req.Body, _ = req.GetBody() // rewind
	}

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
