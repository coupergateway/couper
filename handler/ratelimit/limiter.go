package ratelimit

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/coupergateway/couper/errors"
)

type Limiter struct {
	check     chan *slowTrip
	limits    RateLimits
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
			mode, timeToWait := l.checkCapacity()
			if mode == modeBlock && timeToWait > 0 {
				// We do not wait, we want block directly.
				trip.err = errors.BetaBackendRateLimitExceeded
				trip.out <- trip
			} else {
				select {
				case <-time.After(timeToWait):
					// Noop if 'timeToWait' is 0.
				case <-trip.req.Context().Done():
					// The request was canceled while in the queue.
					trip.err = trip.req.Context().Err()
					trip.out <- trip

					// Do not sleep for X canceled requests.
					continue
				}

				l.countRequest()

				// Do not wait for the response...
				go func() {
					trip.res, trip.err = l.transport.RoundTrip(trip.req)
					trip.out <- trip
				}()
			}
		}
	}
}

func (l *Limiter) checkCapacity() (mode int, t time.Duration) {
	now := time.Now()

	for _, rl := range l.limits {
		if rl.getPeriodStart().IsZero() {
			rl.setPeriodStart(now)
		}

		switch rl.window {
		case windowFixed:
			// Update current period.
			currentPeriod := rl.getPeriodStart()
			multiplicator := ((now.UnixNano() - currentPeriod.UnixNano()) / int64(time.Nanosecond)) / rl.period.Nanoseconds()
			if multiplicator > 0 {
				currentPeriod = currentPeriod.Add(time.Duration(rl.period.Nanoseconds() * multiplicator))
				rl.setPeriodStart(currentPeriod)
				rl.count.Store(0)
			}

			//fmt.Printf("Period start: %v, Current time: %v, Count: %d\n", currentPeriod, now, rl.count.Load()) // debug

			if rl.count.Load() >= rl.perPeriod {
				// Calculate the 'timeToWait'.
				t = time.Duration((currentPeriod.Add(rl.period).UnixNano() - now.UnixNano()) / int64(time.Nanosecond))

				mode = rl.mode
			}
		case windowSliding:
			latest := rl.ringBuffer.get()

			if !latest.IsZero() && latest.Add(rl.period).After(now) {
				// Calculate the 'timeToWait'.
				t = time.Duration((latest.Add(rl.period).UnixNano() - now.UnixNano()) / int64(time.Nanosecond))

				mode = rl.mode
			}
			// no default: config validation ensures that only 'windowFixed' and 'windowSliding' are possible
		}
	}

	return mode, t
}

// countRequest MUST only be called after checkCapacity
func (l *Limiter) countRequest() {
	for _, rl := range l.limits {
		switch rl.window {
		case windowFixed:
			rl.count.Add(1)
		case windowSliding:
			rl.ringBuffer.put(time.Now())
		}
	}
}
