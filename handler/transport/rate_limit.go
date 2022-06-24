package transport

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
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
	window      int
	quitCh      <-chan struct{} // GC
}

type RateLimits []*RateLimit

type Limiter struct {
	check     chan chan bool
	limits    RateLimits
	transport http.RoundTripper
}

func NewLimiter(transport http.RoundTripper, limits RateLimits) *Limiter {
	limiter := &Limiter{
		check:     make(chan chan bool),
		limits:    limits,
		transport: transport,
	}

	for _, rl := range limiter.limits {
		go rl.gc(time.Second)
	}

	go limiter.checkCapacity()

	return limiter
}

func (l *Limiter) checkCapacity() {
	// TODO Shutdown
outer:
	for {
		result := <-l.check

		for _, rl := range l.limits {
			if !rl.hasCapacity() {
				result <- false

				continue outer
			}
		}

		result <- true
	}
}

func (l *Limiter) RoundTrip(req *http.Request) (*http.Response, error) {
	resultCh := make(chan bool)

	select {
	case l.check <- resultCh:
		select {
		case result := <-resultCh:
			if result {
				res, err := l.transport.RoundTrip(req)
				if res != nil && res.StatusCode == http.StatusTooManyRequests {
					return res, errors.Backend.Status(http.StatusTooManyRequests).With(err)
				}

				return res, err
			} else {
				return nil, errors.Backend.Status(http.StatusTooManyRequests)
			}
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
}

func ConfigureRateLimits(ctx context.Context, limits config.RateLimits, logger *logrus.Entry) (RateLimits, error) {
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

		rateLimits = append(rateLimits, rateLimit)
	}

	// Sort 'rateLimits' by 'period' DESC.
	sort.Slice(rateLimits, func(i, j int) bool {
		return rateLimits[i].period > rateLimits[j].period
	})

	return rateLimits, nil
}

func (rl *RateLimit) hasCapacity() bool {
	now := time.Now()

	rl.mu.Lock()

	switch rl.window {
	case windowFixed:
		// TODO
	case windowSliding:
		if diff := rl.periodEnd.Sub(rl.periodStart); diff < rl.period {
			// First, not full period
			scaleFactor := (int)(diff / (rl.period / 100))
			if scaleFactor == 0 {
				scaleFactor = 1
			}

			maxAllowedRequests := (int)(rl.perPeriod / 100 * uint(scaleFactor))
			if maxAllowedRequests == 0 {
				maxAllowedRequests = 1
			}

			fmt.Printf(">>> %#v\n", scaleFactor)
			fmt.Printf(">>> %#v\n", maxAllowedRequests)

			if len(rl.counter) > maxAllowedRequests {
				return false
			}

			rl.counter = append(rl.counter, now)

			return true
		} else {
			var done uint
			var last = now.Add(-1 * rl.period)

			for i := len(rl.counter) - 1; i >= 0; i-- {
				if rl.counter[i].Before(last) {
					break
				}

				done++
			}

			if done > rl.perPeriod {
				return false
			}

			rl.counter = append(rl.counter, now)

			return true
		}
	}

	rl.mu.Unlock()

	return true
}

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
					rl.periodStart = rl.periodEnd.Add(interval)
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
