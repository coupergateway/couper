package config

// Throttle represents the <config.Throttle> object.
type Throttle struct {
	Mode         string `hcl:"mode,optional" default:"wait" docs:"If {mode} is set to {block} and the throttle limit is exceeded, the client request is immediately answered with HTTP status code {429} (Too Many Requests) and no backend request is made. If {mode} is set to {wait} and the throttle limit is exceeded, the request waits for the next free throttling period."`
	Period       string `hcl:"period" docs:"Defines the throttle period." type:"duration"`
	PerPeriod    uint64 `hcl:"per_period" docs:"Defines the number of allowed backend requests in a period."`
	PeriodWindow string `hcl:"period_window,optional" default:"sliding" docs:"Defines the window of the period. A {fixed} window permits {per_period} requests within {period} after the first request to the parent backend. After the {period} has expired, another {per_period} request is permitted. The sliding window ensures that only {per_period} requests are sent in any interval of length {period}."`
}

// Throttles represents a list of <config.Throttle> objects.
type Throttles []*Throttle
