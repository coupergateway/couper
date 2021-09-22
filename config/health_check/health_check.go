package health_check

import "time"

var defaultHealthCheck = &ParsedHealthCheck{
	FailureThreshold: 5,
	Period:           time.Second,
	Timeout:          time.Second,
}

type ParsedHealthCheck struct {
	FailureThreshold int
	Period           time.Duration
	Timeout          time.Duration
}

type HealthCheck struct {
	FailureThreshold int    `hcl:"failure_threshold,optional"`
	Period           string `hcl:"period,optional"`
	Timeout          string `hcl:"timeout,optional"`
}

func (target *ParsedHealthCheck) Parse(health *HealthCheck) (err error) {
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
	if target.FailureThreshold == 0 {
		target.FailureThreshold = defaultHealthCheck.FailureThreshold
	}
	return nil
}
