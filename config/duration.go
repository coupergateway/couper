package config

import (
	"fmt"
	"time"
)

func ParseDuration(attribute string, value string, _default time.Duration) (time.Duration, error) {
	if value == "" {
		return _default, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %s", attribute, err)
	}
	if duration < 0 {
		return 0, fmt.Errorf("%s: cannot be negative: '%s'", attribute, value)
	}

	return duration, nil
}
