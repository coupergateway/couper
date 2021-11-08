package health_check

import (
	"errors"
	"time"
)

var defaultHealthCheck = &ParsedOptions{
	FailureThreshold: 0,
	Interval:         time.Second,
	Timeout:          time.Second,
}

type ParsedOptions struct {
	FailureThreshold int
	Interval         time.Duration
	Timeout          time.Duration
}

type Options struct {
	FailureThreshold int    `hcl:"failure_threshold,optional"`
	Interval         string `hcl:"interval,optional"`
	Timeout          string `hcl:"timeout,optional"`
}

func (target *ParsedOptions) Parse(health *Options) (err error) {
	if health == nil {
		return errors.New("nil pointer dereference")
	}
	if health.Interval == "" {
		target.Interval = defaultHealthCheck.Interval
	} else {
		target.Interval, err = time.ParseDuration(health.Interval)
		if err != nil {
			return err
		}
	}
	if health.Timeout == "" {
		target.Timeout = defaultHealthCheck.Timeout
	} else {
		target.Timeout, err = time.ParseDuration(health.Timeout)
		if err != nil {
			return err
		}
	}
	target.FailureThreshold = health.FailureThreshold
	return nil
}
