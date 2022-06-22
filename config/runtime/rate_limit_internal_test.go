package runtime

import (
	"context"
	"testing"
	"time"

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
		{
			[]*config.RateLimit{
				{Period: &min, PerPeriod: &num, PeriodWindow: "test"},
			},
			`unsupported 'period_window' ("test") given`,
		},
	} {
		_, err := configureRateLimits(context.TODO(), tc.configured, nil)
		if err == nil {
			t.Fatal("Missing error")
		}

		if got := err.Error(); got != tc.expMessage {
			t.Errorf("exp: %q\ngot: %q", tc.expMessage, got)
		}
	}
}

func TestRateLimits_GC_Sliding(t *testing.T) {
	ctx, chancel := context.WithCancel(context.TODO())
	now := time.Now()

	rateLimit := &RateLimit{
		counter: []time.Time{
			now.Add(-7 * time.Second),
			now.Add(-6 * time.Second),
			now.Add(-5 * time.Second),
			now.Add(-4 * time.Second),
			now.Add(-3 * time.Second),
			now.Add(-2 * time.Second),
			now.Add(-1 * time.Second),
		},
		period:      5 * time.Second,
		periodEnd:   now.Add(2 * time.Second),
		periodStart: now.Add(-3 * time.Second),
		window:      windowSliding,
		quitCh:      ctx.Done(),
	}

	go rateLimit.gc(time.Second)

	time.Sleep(1100 * time.Millisecond)
	chancel()

	rateLimit.mu.Lock()
	if l := len(rateLimit.counter); l != 3 {
		t.Errorf("exp: 3\ngot: %d", l)
	}
	rateLimit.mu.Unlock()
}

func TestRateLimits_GC_Fixed(t *testing.T) {
	ctx, chancel := context.WithCancel(context.TODO())
	now := time.Now()

	rateLimit := &RateLimit{
		counter: []time.Time{
			now.Add(-7 * time.Second),
			now.Add(-6 * time.Second),
			now.Add(-5 * time.Second),
			now.Add(-4 * time.Second),
			now.Add(-3 * time.Second),
			now.Add(-2 * time.Second),
			now.Add(-1 * time.Second),
		},
		period:      5 * time.Second,
		periodEnd:   now,
		periodStart: now.Add(-5 * time.Second),
		window:      windowFixed,
		quitCh:      ctx.Done(),
	}

	go rateLimit.gc(time.Second)

	time.Sleep(1100 * time.Millisecond)
	chancel()

	rateLimit.mu.Lock()
	if l := len(rateLimit.counter); l != 0 {
		t.Errorf("exp: 0\ngot: %d", l)
	}
	rateLimit.mu.Unlock()
}
