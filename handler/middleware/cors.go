package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/internal/seetie"
	"github.com/coupergateway/couper/server/writer"
)

var _ http.Handler = &CORS{}

type CORS struct {
	options     *CORSOptions
	nextHandler http.Handler
}

type CORSOptions struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxAge           string
	methodAllowed    methodAllowedFunc
}

func NewCORSOptions(cors *config.CORS, methodAllowed methodAllowedFunc) (*CORSOptions, error) {
	if cors == nil {
		return nil, nil
	}

	var corsMaxAge string
	if cors.MaxAge != "" {
		dur, err := time.ParseDuration(cors.MaxAge)
		if err != nil {
			return nil, errors.Configuration.With(err).Message("cors max_age")
		}
		corsMaxAge = strconv.Itoa(int(math.Floor(dur.Seconds())))
	}

	allowedOrigins := seetie.ValueToStringSlice(cors.AllowedOrigins)

	for i, a := range allowedOrigins {
		allowedOrigins[i] = strings.ToLower(a)
	}

	return &CORSOptions{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: cors.AllowCredentials,
		MaxAge:           corsMaxAge,
		methodAllowed:    methodAllowed,
	}, nil
}

func (c *CORSOptions) AllowsOrigin(origin string) bool {
	if c == nil {
		return false
	}

	for _, a := range c.AllowedOrigins {
		if a == strings.ToLower(origin) || a == "*" {
			return true
		}
	}

	return false
}

func NewCORSHandler(opts *CORSOptions, nextHandler http.Handler) http.Handler {
	if opts == nil {
		return nextHandler
	}
	return &CORS{
		options:     opts,
		nextHandler: nextHandler,
	}
}

func (c *CORS) ServeNextHTTP(rw http.ResponseWriter, nextHandler http.Handler, req *http.Request) {
	if response, ok := rw.(*writer.Response); ok {
		response.AddHeaderModifier(func(header http.Header) {
			c.setCorsRespHeaders(header, req)
		})
	}

	if c.isCorsPreflightRequest(req) {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	nextHandler.ServeHTTP(rw, req)
}

func (c *CORS) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c.ServeNextHTTP(rw, c.nextHandler, req)
}

func (c *CORS) isCorsPreflightRequest(req *http.Request) bool {
	return req.Method == http.MethodOptions &&
		(req.Header.Get("Access-Control-Request-Method") != "" ||
			req.Header.Get("Access-Control-Request-Headers") != "")
}

func (c *CORS) setCorsRespHeaders(headers http.Header, req *http.Request) {
	headers.Del("Access-Control-Allow-Origin")
	headers.Del("Access-Control-Allow-Credentials")
	headers.Del("Access-Control-Allow-Headers")
	headers.Del("Access-Control-Allow-Methods")
	headers.Del("Access-Control-Max-Age")
	// see https://fetch.spec.whatwg.org/#http-responses
	allowSpecificOrigin := false
	if c.options.AllowsOrigin("*") && !c.options.AllowCredentials {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Add("Vary", "Origin")
		allowSpecificOrigin = true
	}

	if !c.isCorsRequest(req) {
		return
	}

	requestOrigin := req.Header.Get("Origin")
	if !c.options.AllowsOrigin(requestOrigin) {
		return
	}

	if allowSpecificOrigin {
		headers.Set("Access-Control-Allow-Origin", requestOrigin)
	}

	if c.options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	if c.isCorsPreflightRequest(req) {
		// Reflect request header value
		acrm := req.Header.Get("Access-Control-Request-Method")
		if acrm != "" {
			if c.options.methodAllowed == nil || c.options.methodAllowed(acrm) {
				headers.Set("Access-Control-Allow-Methods", acrm)
			}
			headers.Add("Vary", "Access-Control-Request-Method")
		}
		// Reflect request header value
		acrh := req.Header.Get("Access-Control-Request-Headers")
		if acrh != "" {
			headers.Set("Access-Control-Allow-Headers", acrh)
			headers.Add("Vary", "Access-Control-Request-Headers")
		}
		if c.options.MaxAge != "" {
			headers.Set("Access-Control-Max-Age", c.options.MaxAge)
		}
	}
}

func (c *CORS) isCorsRequest(req *http.Request) bool {
	return req.Header.Get("Origin") != ""
}

func (c *CORS) Child() http.Handler {
	return c.nextHandler
}
