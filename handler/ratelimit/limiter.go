package ratelimit

import (
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/coupergateway/couper/errors"
)

type Limiter struct {
	check     chan *slowTrip
	limits    RateLimits
	mu        sync.RWMutex
	transport http.RoundTripper
}

// slowTrip is a RoundTrip container for the limiter.
type slowTrip struct {
	err    error
	out    chan *slowTrip
	quitCh <-chan struct{}
	req    *http.Request
	res    *http.Response
}

// NewLimiter creates a new Rate Limiter. See RateLimit for configuration options.
func NewLimiter(transport http.RoundTripper, limits RateLimits) *Limiter {
	if len(limits) == 0 {
		return nil
	}

	limiter := &Limiter{
		check:     make(chan *slowTrip),
		limits:    limits,
		transport: transport,
	}

	go limiter.slowTripper()

	return limiter
}

func (l *Limiter) RoundTrip(req *http.Request) (*http.Response, error) {
	outCh := make(chan *slowTrip)

	trip := &slowTrip{
		out:    outCh,
		quitCh: l.limits[0].quitCh,
		req:    req,
	}

	select {
	case l.check <- trip:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}

	trip = <-outCh

	return trip.res, trip.err
}

func (l *Limiter) slowTripper() {
	defer func() {
		if rc := recover(); rc != nil {
			l.limits[0].logger.WithField("panic", string(debug.Stack())).Panic(rc)
		}
	}()

	for {
		select {
		case <-l.limits[0].quitCh:
			return
		case trip := <-l.check:
			select {
			case <-trip.req.Context().Done():
				// The request was canceled while in the queue.
				trip.err = trip.req.Context().Err()
				trip.out <- trip

				// Do not sleep for X canceled requests.
				continue
			default:
			}

			l.mu.Lock()

			if mode, timeToWait := l.checkCapacity(); mode == modeBlock && timeToWait > 0 {
				// We do not wait, we want block directly.
				trip.err = errors.BetaBackendRateLimitExceeded
				trip.out <- trip

				l.mu.Unlock()
			} else {
				select {
				// Noop if 'timeToWait' is 0.
				case <-time.After(timeToWait):
				case <-trip.req.Context().Done():
					// The request was canceled while in the queue.
					trip.err = trip.req.Context().Err()
					trip.out <- trip

					// Do not sleep for X canceled requests.
					l.mu.Unlock()
					continue
				}

				l.countRequest()

				l.mu.Unlock()

				// Do not wait for the response...
				go func() {
					trip.res, trip.err = l.transport.RoundTrip(trip.req)

					if trip.res != nil && trip.res.StatusCode == http.StatusTooManyRequests {
						trip.err = errors.BetaBackendRateLimitExceeded.With(trip.err)
					}

					trip.out <- trip
				}()
			}
		}
	}
}

func (l *Limiter) checkCapacity() (mode int, t time.Duration) {
	now := time.Now()

	for _, rl := range l.limits {
		if rl.periodStart.IsZero() {
			rl.periodStart = now
		}
		
		switch rl.window {
		case windowFixed:
			// Update current period.
			multiplicator := ((now.UnixNano() - rl.periodStart.UnixNano()) / int64(time.Nanosecond)) / rl.period.Nanoseconds()
			if multiplicator > 0 {
				rl.periodStart = rl.periodStart.Add(time.Duration(rl.period.Nanoseconds() * multiplicator))
				rl.count = 0
			}

			if rl.count >= rl.perPeriod {
				// Calculate the 'timeToWait'.
				t = time.Duration((rl.periodStart.Add(rl.period).UnixNano() - now.UnixNano()) / int64(time.Nanosecond))

				mode = rl.mode
			}
		case windowSliding:
			latest := rl.ringBuffer.get()

			if !latest.IsZero() && latest.Add(rl.period).After(now) {
				// Calculate the 'timeToWait'.
				t = time.Duration((latest.Add(rl.period).UnixNano() - now.UnixNano()) / int64(time.Nanosecond))

				mode = rl.mode
			}
		}
	}

	return
}

// countRequest MUST only be called after checkCapacity()
func (l *Limiter) countRequest() {
	for _, rl := range l.limits {
		rl.countRequest()
	}
}
