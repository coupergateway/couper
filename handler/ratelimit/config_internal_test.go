package ratelimit

import (
	"context"
	"testing"

	"github.com/avenga/couper/config"
)

func TestConfig_Errors(t *testing.T) {
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
			[]*config.BetaRateLimit{
				{PerPeriod: num, PeriodWindow: ""},
			},
			"'period' must not be 0 (zero)",
		},
		{
			[]*config.BetaRateLimit{
				{Period: min, PeriodWindow: ""},
			},
			"'per_period' must not be 0 (zero)",
		},
		{
			[]*config.BetaRateLimit{
				{Period: foo, PerPeriod: num, PeriodWindow: ""},
			},
			`period: time: invalid duration "foo"`,
		},
		{
			[]*config.BetaRateLimit{
				{Period: neg, PerPeriod: num, PeriodWindow: ""},
			},
			`period: cannot be negative: '-1s'`,
		},
		{
			[]*config.BetaRateLimit{
				{Period: zeroStr, PerPeriod: num, PeriodWindow: ""},
			},
			`'period' must not be 0 (zero)`,
		},
		{
			[]*config.BetaRateLimit{
				{Period: min, PerPeriod: zeroInt, PeriodWindow: ""},
			},
			`'per_period' must not be 0 (zero)`,
		},
		{
			[]*config.BetaRateLimit{
				{Period: min, PerPeriod: num, PeriodWindow: ""},
				{Period: sec, PerPeriod: num, PeriodWindow: ""},
			},
			`duplicate period ("60s") found`,
		},
		{
			[]*config.BetaRateLimit{
				{Period: min, PerPeriod: num, PeriodWindow: "test"},
			},
			`unsupported 'period_window' ("test") given`,
		},
		{
			[]*config.BetaRateLimit{
				{Period: min, PerPeriod: num, PeriodWindow: "", Mode: "test"},
			},
			`unsupported 'mode' ("test") given`,
		},
	} {
		_, err := ConfigureRateLimits(context.TODO(), tc.configured, nil)
		if err == nil {
			t.Fatal("Missing error")
		}

		if got := err.Error(); got != tc.expMessage {
			t.Errorf("exp: %q\ngot: %q", tc.expMessage, got)
		}
	}
}
