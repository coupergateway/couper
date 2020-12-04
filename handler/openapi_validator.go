package handler

import (
	"bytes"
	"context"
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
	router                   *openapi3filter.Router
	buffer                   eval.BufferOption
	ignoreRequestViolations  bool
	ignoreResponseViolations bool
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

	apiValidation := eval.BufferRequest | eval.BufferResponse
	if openapi.IgnoreRequestViolations {
		apiValidation ^= eval.BufferRequest
	}
	if openapi.IgnoreResponseViolations {
		apiValidation ^= eval.BufferResponse
	}

	return &OpenAPIValidatorOptions{
		buffer:                   apiValidation,
		router:                   router,
		ignoreRequestViolations:  openapi.IgnoreRequestViolations,
		ignoreResponseViolations: openapi.IgnoreResponseViolations,
	}, nil
}

func (o *OpenAPIValidatorOptions) NewOpenAPIValidator() *OpenAPIValidator {
	return &OpenAPIValidator{
		options:       o,
		validationCtx: context.Background(),
	}
}

type OpenAPIValidator struct {
	options                *OpenAPIValidatorOptions
	route                  *openapi3filter.Route
	requestValidationInput *openapi3filter.RequestValidationInput
	validationCtx          context.Context
}

func (v *OpenAPIValidator) ValidateRequest(req *http.Request, tripInfo *logging.RoundtripInfo) error {
	route, pathParams, _ := v.options.router.FindRoute(req.Method, req.URL)
	v.route = route

	v.requestValidationInput = &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	err := openapi3filter.ValidateRequest(v.validationCtx, v.requestValidationInput)

	if req.GetBody != nil {
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
	if v.route == nil {
		return nil
	}
	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: v.requestValidationInput,
		Status:                 beresp.StatusCode,
		Header:                 beresp.Header.Clone(),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true /* undefined response codes are invalid */},
	}

	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, beresp.Body)
	if err != nil {
		return err
	}
	// reset
	beresp.Body = eval.NewReadCloser(bytes.NewBuffer(buf.Bytes()), beresp.Body)
	responseValidationInput.SetBodyBytes(buf.Bytes())

	err = openapi3filter.ValidateResponse(v.validationCtx, responseValidationInput)
	if err != nil {
		err = fmt.Errorf("response validation: %w", err)
		if !v.options.ignoreResponseViolations {
			return err
		}
		tripInfo.ValidationError = append(tripInfo.ValidationError, err)
	}
	return nil
}
