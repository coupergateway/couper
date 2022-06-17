package runtime

import (
	"testing"

	"github.com/avenga/couper/config"
)

func TestRateLimits_Errors(t *testing.T) {
	type testCase struct {
		configured config.RateLimits
		expMessage string
	}

	var (
		foo     string = "foo"
		min     string = "1m"
		sec     string = "60s"
		neg     string = "-1s"
		num     uint   = 123
		zeroStr string = "0s"
		zeroInt uint   = 0
	)

	for _, tc := range []testCase{
		{
			[]*config.RateLimit{
				{Period: nil, PerPeriod: nil, PeriodWindow: ""},
			},
			"misiing required 'period' attribute",
		},
		{
			[]*config.RateLimit{
				{Period: &min, PerPeriod: nil, PeriodWindow: ""},
			},
			"misiing required 'per_period' attribute",
		},
		{
			[]*config.RateLimit{
				{Period: &foo, PerPeriod: &num, PeriodWindow: ""},
			},
			`period: time: invalid duration "foo"`,
		},
		{
			[]*config.RateLimit{
				{Period: &neg, PerPeriod: &num, PeriodWindow: ""},
			},
			`period: cannot be negative: "-1s"`,
		},
		{
			[]*config.RateLimit{
				{Period: &zeroStr, PerPeriod: &num, PeriodWindow: ""},
			},
			`'period' must not be 0 (zero)`,
		},
		{
			[]*config.RateLimit{
				{Period: &min, PerPeriod: &zeroInt, PeriodWindow: ""},
			},
			`'per_period' must not be 0 (zero)`,
		},
		{
			[]*config.RateLimit{
				{Period: &min, PerPeriod: &num, PeriodWindow: ""},
				{Period: &sec, PerPeriod: &num, PeriodWindow: ""},
			},
			`duplicate period ("60s") found`,
		},
	} {
		_, err := configureRateLimits(tc.configured)
		if err == nil {
			t.Fatal("Missing error")
		}

		if got := err.Error(); got != tc.expMessage {
			t.Errorf("exp: %q\ngot: %q", tc.expMessage, got)
		}
	}
}
