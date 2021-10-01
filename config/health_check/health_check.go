package health_check

import (
	"errors"
	"time"
)

var defaultHealthCheck = &ParsedOptions{
	FailureThreshold: 0,
	Period:           time.Second,
	Timeout:          time.Second,
}

type ParsedOptions struct {
	FailureThreshold int
	Period           time.Duration
	Timeout          time.Duration
}

type Options struct {
	FailureThreshold int    `hcl:"failure_threshold,optional"`
	Period           string `hcl:"period,optional"`
	Timeout          string `hcl:"timeout,optional"`
}

func (target *ParsedOptions) Parse(health *Options) (err error) {
	if health == nil {
		return errors.New("nil pointer dereference")
	}
	if health.Period == "" {
		target.Period = defaultHealthCheck.Period
	} else {
		target.Period, err = time.ParseDuration(health.Period)
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
