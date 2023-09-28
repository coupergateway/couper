package ratelimit

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/coupergateway/couper/config"
	"github.com/sirupsen/logrus"
)

const (
	dummy = iota
	modeBlock
	modeWait
	windowFixed
	windowSliding
)

type RateLimit struct {
	count       uint
	logger      *logrus.Entry
	mode        int
	period      time.Duration
	periodStart time.Time
	perPeriod   uint
	ringBuffer  *ringBuffer
	window      int
	quitCh      <-chan struct{}
}

type RateLimits []*RateLimit

func ConfigureRateLimits(ctx context.Context, limits config.RateLimits, logger *logrus.Entry) (RateLimits, error) {
	var (
		mode       int
		rateLimits RateLimits
		window     int
	)

	uniqueDurations := make(map[time.Duration]struct{})

	for _, limit := range limits {
		d, err := config.ParseDuration("period", limit.Period, 0)
		if err != nil {
			return nil, err
		}

		if d == 0 {
			return nil, fmt.Errorf("'period' must not be 0 (zero)")
		}
		if limit.PerPeriod == 0 {
			return nil, fmt.Errorf("'per_period' must not be 0 (zero)")
		}

		if _, ok := uniqueDurations[time.Duration(d.Nanoseconds())]; ok {
			return nil, fmt.Errorf("duplicate period (%q) found", limit.Period)
		}

		uniqueDurations[time.Duration(d.Nanoseconds())] = struct{}{}

		switch limit.PeriodWindow {
		case "":
			fallthrough
		case "sliding":
			window = windowSliding
		case "fixed":
			window = windowFixed
		default:
			return nil, fmt.Errorf("unsupported 'period_window' (%q) given", limit.PeriodWindow)
		}

		switch limit.Mode {
		case "":
			fallthrough
		case "wait":
			mode = modeWait
		case "block":
			mode = modeBlock
		default:
			return nil, fmt.Errorf("unsupported 'mode' (%q) given", limit.Mode)
		}

		rateLimit := &RateLimit{
			logger:    logger,
			mode:      mode,
			period:    time.Duration(d.Nanoseconds()),
			perPeriod: limit.PerPeriod,
			window:    window,
			quitCh:    ctx.Done(),
		}

		switch rateLimit.window {
		case windowSliding:
			rateLimit.ringBuffer = newRingBuffer(rateLimit.perPeriod)
		}

		rateLimits = append(rateLimits, rateLimit)
	}

	// Sort 'rateLimits' by 'period' DESC.
	sort.Slice(rateLimits, func(i, j int) bool {
		return rateLimits[i].period > rateLimits[j].period
	})

	return rateLimits, nil
}

// countRequest MUST only be called after checkCapacity()
func (rl *RateLimit) countRequest() {
	switch rl.window {
	case windowFixed:
		rl.count++
	case windowSliding:
		rl.ringBuffer.put(time.Now())
	}
}
