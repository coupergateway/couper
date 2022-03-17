package config

import (
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/hcl/v2"
	
	"github.com/avenga/couper/utils"
)

var defaultHealthCheck = &HealthCheck{
	FailureThreshold: 2,
	Interval:         time.Second,
	Timeout:          time.Second,
	ExpectStatus:     map[int]bool{200: true, 204: true, 301: true},
	ExpectText:       "",
}

type HealthCheck struct {
	FailureThreshold uint
	Interval         time.Duration
	Timeout          time.Duration
	ExpectStatus     map[int]bool
	ExpectText       string
	Request          *http.Request
}

type Headers map[string]string

type Health struct {
	FailureThreshold uint     `hcl:"failure_threshold,optional"`
	Interval         string   `hcl:"interval,optional"`
	Timeout          string   `hcl:"timeout,optional"`
	Path             string   `hcl:"path,optional"`
	ExpectStatus     int      `hcl:"expect_status,optional"`
	ExpectText       string   `hcl:"expect_text,optional"`
	Headers          Headers  `hcl:"headers,optional"`
	Remain           hcl.Body `hcl:",remain"`
}

func NewHealthCheck(baseURL string, options *Health) (*HealthCheck, error) {
	healthCheck := *defaultHealthCheck

	var err error
	if options != nil {
		if options.Interval != "" {
			healthCheck.Interval, err = time.ParseDuration(options.Interval)
			if err != nil {
				return nil, err
			}
			healthCheck.Timeout = healthCheck.Interval
		}
		if options.Timeout != "" {
			healthCheck.Timeout, err = time.ParseDuration(options.Timeout)
			if err != nil {
				return nil, err
			}
		}
		if healthCheck.Timeout > healthCheck.Interval {
			healthCheck.Timeout = healthCheck.Interval
		}
		if options.FailureThreshold != 0 {
			healthCheck.FailureThreshold = options.FailureThreshold
		}
		if options.ExpectStatus != 0 {
			healthCheck.ExpectStatus = map[int]bool{options.ExpectStatus: true}
		}
		healthCheck.ExpectText = options.ExpectText

		request, err := http.NewRequest(http.MethodGet, baseURL, nil)
		if err != nil {
			return nil, err
		}

		if options.Path != "" {
			request.URL = request.URL.ResolveReference(createURL(options.Path))
		}

		if options.Headers != nil {
			for key, value := range options.Headers {
				request.Header.Add(key, value)
			}
		}

		if ua := request.Header.Get("User-Agent"); ua == "" {
			request.Header.Set("User-Agent", "Couper / "+utils.VersionName+" health-check")
		}

		healthCheck.Request = request
	}
	return &healthCheck, err
}

func createURL(pathQuery string) *url.URL {
	uri, _ := url.ParseRequestURI("http://HOST/../" + pathQuery)
	uri.Scheme = ""
	uri.Host = ""
	u := url.URL{}
	return u.ResolveReference(uri)
}
