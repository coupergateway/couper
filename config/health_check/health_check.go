package health_check

import (
	"github.com/hashicorp/hcl/v2"
	"time"
)

var defaultHealthCheck = &ParsedOptions{
	FailureThreshold: 2,
	Interval:         time.Second,
	Timeout:          time.Second,
}

type ParsedOptions struct {
	FailureThreshold uint
	Interval         time.Duration
	Timeout          time.Duration
}

type Options struct {
	FailureThreshold uint     `hcl:"failure_threshold,optional"`
	Interval         string   `hcl:"interval,optional"`
	Timeout          string   `hcl:"timeout,optional"`
	Remain           hcl.Body `hcl:",remain"`
}

func NewHealthCheck(options *Options) (*ParsedOptions, error) {
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
	}
	return &healthCheck, err
}
