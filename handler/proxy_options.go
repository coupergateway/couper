package handler

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/docker/go-units"
	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
)

type ProxyOptions struct {
	BasicAuth        string
	Context          hcl.Body
	CORS             *CORSOptions
	ErrorTemplate    *errors.Template
	Kind             string
	OpenAPI          *OpenAPIValidatorOptions
	RequestBodyLimit int64
	Transport        *TransportConfig
}

func NewProxyOptions(
	conf *config.Backend, corsOpts *CORSOptions, noProxyFromEnv bool,
	errTpl *errors.Template, kind string,
) (*ProxyOptions, error) {
	var timeout, connTimeout, ttfbTimeout time.Duration
	if err := parseDuration(conf.Timeout, &timeout); err != nil {
		return nil, err
	}
	if err := parseDuration(conf.TTFBTimeout, &ttfbTimeout); err != nil {
		return nil, err
	}
	if err := parseDuration(conf.ConnectTimeout, &connTimeout); err != nil {
		return nil, err
	}

	bodyLimit, err := units.FromHumanSize(conf.RequestBodyLimit)
	if err != nil {
		return nil, fmt.Errorf("backend bodyLimit: %v", err)
	}

	cors := corsOpts
	if cors == nil { // Could be nil on non api context like 'free' endpoints or definitions.
		cors = &CORSOptions{}
	}

	openAPIValidatorOptions, err := NewOpenAPIValidatorOptions(conf.OpenAPI)
	if err != nil {
		return nil, err
	}

	return &ProxyOptions{
		BasicAuth:        conf.BasicAuth,
		CORS:             cors,
		Context:          conf.Remain,
		OpenAPI:          openAPIValidatorOptions,
		ErrorTemplate:    errTpl,
		Kind:             kind,
		RequestBodyLimit: bodyLimit,
		Transport: &TransportConfig{
			BackendName:            conf.Name,
			ConnectTimeout:         connTimeout,
			DisableCertValidation:  conf.DisableCertValidation,
			DisableConnectionReuse: conf.DisableConnectionReuse,
			HTTP2:                  conf.HTTP2,
			MaxConnections:         conf.MaxConnections,
			NoProxyFromEnv:         noProxyFromEnv,
			Proxy:                  conf.Proxy,
			TTFBTimeout:            ttfbTimeout,
			Timeout:                timeout,
		},
	}, nil
}

func (po *ProxyOptions) Hash() string {
	h := sha256.New()
	// exclude hcl list
	opts := *po
	opts.Context = nil
	h.Write([]byte(fmt.Sprintf("%v", opts)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// parseDuration sets the target value if the given duration string is not empty.
func parseDuration(src string, target *time.Duration) error {
	d, err := time.ParseDuration(src)
	if src != "" && err != nil {
		return err
	}
	*target = d
	return nil
}
