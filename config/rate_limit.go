package config

// RateLimit represents the <config.RateLimit> object.
type RateLimit struct {
	Mode         string `hcl:"mode,optional"`
	Period       string `hcl:"period"`
	PerPeriod    uint   `hcl:"per_period"`
	PeriodWindow string `hcl:"period_window,optional"`
}

// RateLimits represents a list of <config.RateLimits> objects.
type RateLimits []*RateLimit
