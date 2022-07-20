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
	dummy = iota
	modeBlock
	modeWait
	windowFixed
	windowSliding
)

type ringBuffer struct {
	buf []time.Time
	len uint
	mu  sync.RWMutex
	r   uint
	w   uint
}

// newRingBuffer creates a new ringBuffer
// instance. ringBuffer is thread safe.
func newRingBuffer(len uint) *ringBuffer {
	return &ringBuffer{
		buf: make([]time.Time, len),
		len: len,
		r:   0,
		w:   len - 1,
	}
}

// put rotates the ring buffer and puts t at
// the "last" position. r must not be empty.
func (r *ringBuffer) put(t time.Time) {
	if r == nil {
		panic("r must not be empty")
	}

	r.mu.Lock()

	r.r++
	r.r %= r.len

	r.w++
	r.w %= r.len

	r.buf[r.w] = t

	r.mu.Unlock()
}

// get returns the value of the "first" element
// in the ring buffer. r must not be empty.
func (r *ringBuffer) get() time.Time {
	if r == nil {
		panic("r must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.buf[r.r]
}

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

type Limiter struct {
	check     chan *slowTrip
	limits    RateLimits
	mu        sync.RWMutex
	transport http.RoundTripper
}

type slowTrip struct {
	err    error
	out    chan *slowTrip
	quitCh <-chan struct{}
	req    *http.Request
	res    *http.Response
}

func NewLimiter(transport http.RoundTripper, limits RateLimits) *Limiter {
	if len(limits) == 0 {
		return nil
	}

	limiter := &Limiter{
		check:     make(chan *slowTrip),
		limits:    limits,
		transport: transport,
	}

	for _, rl := range limits {
		// Init the start of a period.
		rl.periodStart = time.Now()
	}

	// FIXME: Configure parallelism like:
	// for i = 0; i < max_connection; i++ { go limiter.slowTripper() }
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

func ConfigureRateLimits(ctx context.Context, limits config.RateLimits, logger *logrus.Entry) (RateLimits, error) {
	var (
		mode       int
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
			perPeriod: *limit.PerPeriod,
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
					continue
				}

				l.countRequest()

				l.mu.Unlock()

				trip.res, trip.err = l.transport.RoundTrip(trip.req)

				if trip.res != nil && trip.res.StatusCode == http.StatusTooManyRequests {
					trip.err = errors.BetaBackendRateLimitExceeded.With(trip.err)
				}

				trip.out <- trip
			}
		}
	}
}

func (l *Limiter) checkCapacity() (mode int, t time.Duration) {
	now := time.Now()

	for _, rl := range l.limits {
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

// countRequest MUST only be called after checkCapacity()
func (rl *RateLimit) countRequest() {
	switch rl.window {
	case windowFixed:
		rl.count++
	case windowSliding:
		rl.ringBuffer.put(time.Now())
	}
}
