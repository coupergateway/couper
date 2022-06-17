package runtime

import (
	"fmt"
	"time"

	"github.com/avenga/couper/config"
)

type RateLimit struct {
	// TODO
}

type RateLimits []*RateLimit

func configureRateLimits(limits config.RateLimits) (RateLimits, error) {
	var rateLimits RateLimits

	uniqueDurations := make(map[time.Duration]struct{})

	for _, limit := range limits {
		if limit.Period == nil {
			return nil, fmt.Errorf("misiing required 'period' attribute")
		}
		if limit.PerPeriod == nil {
			return nil, fmt.Errorf("misiing required 'per_period' attribute")
		}

		d, err := config.ParseDuration("period", *limit.Period, 0)
		if err != nil {
			return nil, err
		}
		if d == 0 {
			return nil, fmt.Errorf("'period' must not be 0 (zero)")
		}
		if *limit.PerPeriod == 0 {
			return nil, fmt.Errorf("'per_period' must not be 0 (zero)")
		}

		if _, ok := uniqueDurations[time.Duration(d.Nanoseconds())]; ok {
			return nil, fmt.Errorf("duplicate period (%q) found", *limit.Period)
		}

		uniqueDurations[time.Duration(d.Nanoseconds())] = struct{}{}
	}

	return rateLimits, nil
}
