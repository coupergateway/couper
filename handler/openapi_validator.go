package handler

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/avenga/couper/config"
	"github.com/getkin/kin-openapi/openapi3filter"
)

type OpenAPIValidatorFactory struct {
	router                   *openapi3filter.Router
	ignoreRequestViolations  bool
	ignoreResponseViolations bool
}

func NewOpenAPIValidatorFactory(openapi *config.OpenAPI) (*OpenAPIValidatorFactory, error) {
	if openapi == nil {
		return nil, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	router := openapi3filter.NewRouter()
	err = router.AddSwaggerFromFile(dir + "/" + openapi.File)
	if err != nil {
		return nil, err
	}
	return &OpenAPIValidatorFactory{
		router:                   router,
		ignoreRequestViolations:  openapi.IgnoreRequestViolations,
		ignoreResponseViolations: openapi.IgnoreResponseViolations,
	}, nil
}

func (f *OpenAPIValidatorFactory) NewOpenAPIValidator() *OpenAPIValidator {
	return &OpenAPIValidator{
		factory:       f,
		validationCtx: context.Background(),
	}
}

type OpenAPIValidator struct {
	factory                *OpenAPIValidatorFactory
	route                  *openapi3filter.Route
	requestValidationInput *openapi3filter.RequestValidationInput
	validationCtx          context.Context
	Body                   []byte
}

func (v *OpenAPIValidator) ValidateRequest(req *http.Request) (bool, error) {
	route, pathParams, _ := v.factory.router.FindRoute(req.Method, req.URL)
	v.route = route

	v.requestValidationInput = &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	return v.factory.ignoreRequestViolations, openapi3filter.ValidateRequest(v.validationCtx, v.requestValidationInput)
}

func (v *OpenAPIValidator) ValidateResponse(res *http.Response) (bool, error) {
	if v.route == nil {
		return v.factory.ignoreResponseViolations, nil
	}
	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: v.requestValidationInput,
		Status:                 res.StatusCode,
		Header:                 res.Header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true /* undefined response codes are invalid */},
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return v.factory.ignoreResponseViolations, err
	}
	responseValidationInput.SetBodyBytes(body)
	v.Body = body

	err = openapi3filter.ValidateResponse(v.validationCtx, responseValidationInput)
	if err != nil {
		return v.factory.ignoreResponseViolations, err
	}
	return v.factory.ignoreResponseViolations, nil
}
