package validation

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/avenga/couper/eval"
)

type OpenAPI struct {
	options                *OpenAPIOptions
	requestValidationInput *openapi3filter.RequestValidationInput
}

func NewOpenAPI(opts *OpenAPIOptions) *OpenAPI {
	if opts == nil {
		return nil
	}
	return &OpenAPI{
		options: opts,
	}
}

func (v *OpenAPI) ValidateRequest(req *http.Request) error {
	route, pathParams, err := v.options.router.FindRoute(req.Method, req.URL)
	if err != nil {
		err = fmt.Errorf("request validation: '%s %s': %w", req.Method, req.URL.Path, err)
		if !v.options.ignoreRequestViolations {
			return err
		}
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
	}

	return nil
}

func (v *OpenAPI) ValidateResponse(beresp *http.Response) error {
	// since a request validation could fail and ignored due to user options, the input route MAY be nil
	if v.requestValidationInput == nil || v.requestValidationInput.Route == nil {
		err := fmt.Errorf("response validation: '%s %s': invalid route", beresp.Request.Method, beresp.Request.URL.Path)
		if v.options.ignoreResponseViolations {
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
	}

	return nil
}
