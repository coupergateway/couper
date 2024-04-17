package config

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/coupergateway/couper/utils"
)

var defaultHealthCheck = &HealthCheck{
	FailureThreshold: 2,
	Interval:         time.Second,
	Timeout:          time.Second,
	ExpectedStatus:   map[int]bool{200: true, 204: true, 301: true},
	ExpectedText:     "",
}

type HealthCheck struct {
	Context          context.Context
	ExpectedStatus   map[int]bool
	ExpectedText     string
	FailureThreshold uint
	Interval         time.Duration
	Request          *http.Request
	RequestUIDFormat string
	Timeout          time.Duration
}

type Headers map[string]string

type Health struct {
	FailureThreshold *uint    `hcl:"failure_threshold,optional" docs:"Failed checks needed to consider backend unhealthy." default:"2"`
	Interval         string   `hcl:"interval,optional" docs:"Time interval for recheck." default:"1s"`
	Timeout          string   `hcl:"timeout,optional" docs:"Maximum allowed time limit which is	bounded by {interval}." default:"1s"`
	Path             string   `hcl:"path,optional" docs:"URL path with query on backend host."`
	ExpectedStatus   []int    `hcl:"expected_status,optional" docs:"One of wanted response status codes." default:"[200, 204, 301]"`
	ExpectedText     string   `hcl:"expected_text,optional" docs:"Text which the response body must contain."`
	Headers          Headers  `hcl:"headers,optional" docs:"Request HTTP header fields."`
	Remain           hcl.Body `hcl:",remain"`
}

func NewHealthCheck(baseURL string, options *Health, conf *Couper) (*HealthCheck, error) {
	healthCheck := *defaultHealthCheck

	healthCheck.Context = conf.Context
	healthCheck.RequestUIDFormat = conf.Settings.RequestIDFormat

	if options == nil {
		return &healthCheck, nil
	}
	var err error

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
	if options.FailureThreshold != nil {
		healthCheck.FailureThreshold = *options.FailureThreshold
	}
	if len(options.ExpectedStatus) > 0 {
		statusList := map[int]bool{}
		for _, status := range options.ExpectedStatus {
			statusList[status] = true
		}
		healthCheck.ExpectedStatus = statusList
	}
	healthCheck.ExpectedText = options.ExpectedText

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

	return &healthCheck, err
}

func createURL(pathQuery string) *url.URL {
	uri, _ := url.ParseRequestURI("http://HOST/../" + pathQuery)
	uri.Scheme = ""
	uri.Host = ""
	u := url.URL{}
	return u.ResolveReference(uri)
}
