package handler

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/internal/seetie"
)

type CORSOptions struct {
	AllowedOrigins   []string
	AllowCredentials bool
	Disable          bool
	MaxAge           string
}

func NewCORSOptions(cors *config.CORS) (*CORSOptions, error) {
	if cors == nil {
		return nil, nil
	}

	dur, err := time.ParseDuration(cors.MaxAge)
	if err != nil {
		return nil, err
	}

	corsMaxAge := strconv.Itoa(int(math.Floor(dur.Seconds())))
	allowedOrigins := seetie.ValueToStringSlice(cors.AllowedOrigins)

	for i, a := range allowedOrigins {
		allowedOrigins[i] = strings.ToLower(a)
	}

	return &CORSOptions{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: cors.AllowCredentials,
		Disable:          cors.Disable,
		MaxAge:           corsMaxAge,
	}, nil
}

// NeedsVary if a request with not allowed origin is ignored.
func (c *CORSOptions) NeedsVary() bool {
	if c == nil {
		return false
	}

	return !c.AllowsOrigin("*")
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

func setCorsRespHeaders(cors *CORSOptions, headers http.Header, req *http.Request) {
	if cors == nil || req == nil || !isCorsRequest(req) {
		return
	}

	requestOrigin := req.Header.Get("Origin")
	if !cors.AllowsOrigin(requestOrigin) {
		return
	}

	// see https://fetch.spec.whatwg.org/#http-responses
	if cors.AllowsOrigin("*") && !isCredentialed(req.Header) {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", requestOrigin)
	}

	if cors.AllowCredentials == true {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	if isCorsPreflightRequest(req) {
		// Reflect request header value
		acrm := req.Header.Get("Access-Control-Request-Method")
		if acrm != "" {
			headers.Set("Access-Control-Allow-Methods", acrm)
		}
		// Reflect request header value
		acrh := req.Header.Get("Access-Control-Request-Headers")
		if acrh != "" {
			headers.Set("Access-Control-Allow-Headers", acrh)
		}
		if cors.MaxAge != "" {
			headers.Set("Access-Control-Max-Age", cors.MaxAge)
		}
	} else if cors.NeedsVary() {
		headers.Add("Vary", "Origin")
	}
}

func isCorsRequest(req *http.Request) bool {
	return req.Header.Get("Origin") != ""
}

func isCorsPreflightRequest(req *http.Request) bool {
	return req.Method == http.MethodOptions && (req.Header.Get("Access-Control-Request-Method") != "" || req.Header.Get("Access-Control-Request-Headers") != "")
}

func isCredentialed(headers http.Header) bool {
	return headers.Get("Cookie") != "" || headers.Get("Authorization") != "" || headers.Get("Proxy-Authorization") != ""
}
