package runtime

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/avenga/couper/config"
	"github.com/sirupsen/logrus"
)

const (
	windowFixed = iota
	windowSliding
)

type RateLimit struct {
	counter     []time.Time
	logger      *logrus.Entry
	mu          sync.RWMutex
	period      time.Duration
	periodEnd   time.Time
	periodStart time.Time
	perPeriod   uint
	// request     <-chan *http.Request
	// response    chan<- *http.Response
	window int
	// queue       []*http.Request
	quitCh <-chan struct{} // GC
}

type RateLimits []*RateLimit

func (rl *RateLimit) gc(interval time.Duration) {
	ticker := time.NewTicker(interval)

	defer func() {
		if rc := recover(); rc != nil {
			rl.logger.WithField("panic", string(debug.Stack())).Panic(rc)
		}

		ticker.Stop()
	}()

	for {
		select {
		case <-rl.quitCh:
			return
		case now := <-ticker.C:
			rl.mu.Lock()

			switch rl.window {
			case windowFixed:
				for !rl.periodEnd.After(now) {
					rl.periodStart = rl.periodEnd.Add(1 * interval)
					rl.periodEnd = rl.periodStart.Add(rl.period)
					rl.counter = []time.Time{}
				}
			case windowSliding:
				rl.periodEnd = now

				if rl.periodEnd.Sub(rl.periodStart) > rl.period {
					rl.periodStart = rl.periodEnd.Add(-1 * rl.period)
				}

				for _, t := range rl.counter {
					if t.Before(rl.periodStart) {
						rl.counter = rl.counter[1:]

						continue
					}

					break
				}
			}

			rl.mu.Unlock()
		}
	}
}

func configureRateLimits(ctx context.Context, limits config.RateLimits, logger *logrus.Entry) (RateLimits, error) {
	var (
		rateLimits RateLimits
		window     int
	)

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

		rateLimit := &RateLimit{
			logger:      logger,
			period:      time.Duration(d.Nanoseconds()),
			periodStart: time.Now(),
			perPeriod:   *limit.PerPeriod,
			window:      window,
			quitCh:      ctx.Done(),
		}

		switch rateLimit.window {
		case windowFixed:
			rateLimit.periodEnd = rateLimit.periodStart.Add(rateLimit.period)
		case windowSliding:
			rateLimit.periodEnd = rateLimit.periodStart
		}

		go rateLimit.gc(time.Second)

		rateLimits = append(rateLimits, rateLimit)
	}

	// Sort 'rateLimits' by 'period' DESC.
	sort.Slice(rateLimits, func(i, j int) bool {
		return rateLimits[i].period > rateLimits[j].period
	})

	return rateLimits, nil
}
