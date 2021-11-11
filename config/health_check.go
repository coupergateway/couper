package config

import (
	"github.com/hashicorp/hcl/v2"
	"net/url"
	"time"
)

var defaultHealthCheck = &HealthCheck{
	FailureThreshold: 2,
	Interval:         time.Second,
	Timeout:          time.Second,
	Path:             nil,
	ExpectStatus:     map[int]bool{200: true, 204: true, 301: true},
	ExpectText:       "",
}

type HealthCheck struct {
	FailureThreshold uint
	Interval         time.Duration
	Timeout          time.Duration
	Path             *url.URL
	ExpectStatus     map[int]bool
	ExpectText       string
}

type Health struct {
	FailureThreshold uint     `hcl:"failure_threshold,optional"`
	Interval         string   `hcl:"interval,optional"`
	Timeout          string   `hcl:"timeout,optional"`
	Path             string   `hcl:"path,optional"`
	ExpectStatus     int      `hcl:"expect_status,optional"`
	ExpectText       string   `hcl:"expect_text,optional"`
	Remain           hcl.Body `hcl:",remain"`
}

func NewHealthCheck(options *Health) (*HealthCheck, error) {
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
		if options.Path != "" {
			healthCheck.Path = createURL(options.Path)
		}
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
