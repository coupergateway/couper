package validation

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
)

var swaggers sync.Map

type OpenAPI struct {
	options *OpenAPIOptions
}

func NewOpenAPI(opts *OpenAPIOptions) *OpenAPI {
	if opts == nil {
		return nil
	}
	return &OpenAPI{
		options: opts,
	}
}

func (v *OpenAPI) getModifiedSwagger(key, origin string) (*openapi3.Swagger, error) {
	swagger, exists := swaggers.Load(key)
	if !exists {
		clonedSwagger := cloneSwagger(v.options.swagger)

		var newServers []string
		for _, s := range clonedSwagger.Servers {
			// Do not touch a url if template variables are used.
			if names, err := s.ParameterNames(); len(names) > 0 || err != nil {
				continue
			}

			su, err := url.Parse(s.URL)
			if err != nil {
				return nil, err
			}

			if !su.IsAbs() {
				newServers = append(newServers, origin+s.URL)
				continue
			}

			if su.Port() == "" && (su.Scheme == "https" || su.Scheme == "http") {
				su.Host = su.Hostname() + ":"
				if su.Scheme == "https" {
					su.Host += "443"
				} else {
					su.Host += "80"
				}
				s.URL = su.String()
			}

		}

		for _, ns := range newServers {
			clonedSwagger.AddServer(&openapi3.Server{URL: ns})
		}

		swaggers.Store(key, clonedSwagger)
		swagger = clonedSwagger
	}

	if s, ok := swagger.(*openapi3.Swagger); ok {
		return s, nil
	}

	return nil, fmt.Errorf("swagger wrong type: %v", swagger)
}

func cloneSwagger(s *openapi3.Swagger) *openapi3.Swagger {
	sw := *s
	// this is not a deep clone; we only want to add servers
	sw.Servers = s.Servers[:]
	return &sw
}

func (v *OpenAPI) ValidateRequest(req *http.Request, key string) (*openapi3filter.RequestValidationInput, error) {
	// reqURL is modified due to origin transport configuration
	serverURL := *req.URL
	// possible hostname attribute override
	if _, p, _ := net.SplitHostPort(req.Host); p != "" { // hostname could contain a port already
		serverURL.Host = req.Host
	} else {
		serverURL.Host = req.Host + ":" + serverURL.Port()
	}
	swagger, err := v.getModifiedSwagger(key, serverURL.Scheme+"://"+serverURL.Host)
	if err != nil {
		if ctx, ok := req.Context().Value(request.OpenAPI).(*OpenAPIContext); ok {
			ctx.errors = append(ctx.errors, err)
		}
		if !v.options.ignoreRequestViolations {
			return nil, err
		}
		return nil, nil
	}

	router := openapi3filter.NewRouter()
	if err = router.AddSwagger(swagger); err != nil {
		return nil, err
	}

	route, pathParams, err := router.FindRoute(req.Method, &serverURL)
	if err != nil {
		err = fmt.Errorf("'%s %s': %w", req.Method, req.URL.Path, err)
		if ctx, ok := req.Context().Value(request.OpenAPI).(*OpenAPIContext); ok {
			ctx.errors = append(ctx.errors, err)
		}
		if !v.options.ignoreRequestViolations {
			return nil, err
		}
		return nil, nil
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Options:     v.options.filterOptions,
		PathParams:  pathParams,
		QueryParams: req.URL.Query(),
		Request:     req,
		Route:       route,
	}

	// openapi3filter.ValidateRequestBody also handles resetting the req body after reading until EOF.
	if err = openapi3filter.ValidateRequest(req.Context(), requestValidationInput); err != nil {
		if ctx, ok := req.Context().Value(request.OpenAPI).(*OpenAPIContext); ok {
			ctx.errors = append(ctx.errors, err)
		}
		if !v.options.ignoreRequestViolations {
			return requestValidationInput, err
		}
	}

	return requestValidationInput, nil
}

func (v *OpenAPI) ValidateResponse(beresp *http.Response, requestValidationInput *openapi3filter.RequestValidationInput) error {
	// since a request validation could fail and ignored due to user options, the input route MAY be nil
	if requestValidationInput == nil || requestValidationInput.Route == nil {
		err := fmt.Errorf("'%s %s': invalid route", beresp.Request.Method, beresp.Request.URL.Path)
		if beresp.Request != nil {
			if ctx, ok := beresp.Request.Context().Value(request.OpenAPI).(*OpenAPIContext); ok {
				ctx.errors = append(ctx.errors, err)
			}
		}
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
		RequestValidationInput: requestValidationInput,
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
		if beresp.Request != nil {
			if ctx, ok := beresp.Request.Context().Value(request.OpenAPI).(*OpenAPIContext); ok {
				ctx.errors = append(ctx.errors, err)
			}
		}
		if !v.options.ignoreResponseViolations {
			return err
		}
	}

	return nil
}
