package throttle_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coupergateway/couper/config"
	couperErrors "github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/handler/throttle"
	"github.com/coupergateway/couper/internal/test"
)

func TestLimiter_Sliding(t *testing.T) {
	helper := test.New(t)
	logger, _ := test.NewLogger()

	ctx, cancelFn := context.WithCancel(context.Background())
	t.Cleanup(cancelFn)

	const period = "2s"

	limits, err := throttle.ConfigureThrottles(ctx, config.Throttles{
		&config.Throttle{
			Mode:         "wait",
			Period:       period,
			PerPeriod:    1,
			PeriodWindow: "sliding",
		},
	}, logger.WithContext(ctx))
	helper.Must(err)

	backend := &http.Transport{
		MaxConnsPerHost: 1,
	}

	limiter := throttle.NewLimiter(backend, nil)
	if limiter != nil {
		t.Errorf("expected nil Limiter, got %v", limiter)
	}

	limiter = throttle.NewLimiter(backend, limits)
	if limiter == nil {
		t.Errorf("expected configured Limiter, got %v", limiter)
	}

	t.Run("successful requests", func(st *testing.T) {
		const maxRequests = 4
		const expectedDuration = (2 * time.Second) * (maxRequests - 1)
		stHelper := test.New(st)

		reqCounter := &atomic.Int32{}
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			reqCounter.Add(1)
		}))
		defer origin.Close()
		rctx, tCancel := context.WithCancel(ctx)
		defer tCancel()
		st.Cleanup(func() {
			tCancel()
			origin.Close()
		})

		req, rerr := http.NewRequest(http.MethodGet, origin.URL, nil)
		stHelper.Must(rerr)

		startTime := time.Now()
		wg := sync.WaitGroup{}
		wg.Add(maxRequests)
		for i := 0; i < maxRequests; i++ {
			go func() {
				defer wg.Done()
				resp, e := limiter.RoundTrip(req.WithContext(rctx))
				if e != nil {
					st.Errorf("unexpected error: %v", e)
					return
				}
				if resp.StatusCode != http.StatusOK {
					st.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
				}
			}()
		}
		wg.Wait()
		duration := time.Since(startTime)

		if reqCounter.Load() != maxRequests {
			st.Errorf("expected %d requests, got %d", maxRequests, reqCounter.Load())
		}

		if !fuzzyEqual(duration, expectedDuration, time.Millisecond*100) {
			st.Errorf("expected duration around %v, got %v", expectedDuration, duration)
		}
		st.Logf("duration: %v, expected: %v", duration, expectedDuration)
	})

	// Note: This test does not hit (reproducible) the coverage of the limiter until the limiter is using a buffered channel.
	t.Run("canceled request", func(st *testing.T) {
		const maxRequests = 4
		stHelper := test.New(st)

		reqCounter := &atomic.Int32{}
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			reqCounter.Add(1)
		}))
		st.Cleanup(func() { origin.Close() })

		req, rerr := http.NewRequest(http.MethodGet, origin.URL, nil)
		stHelper.Must(rerr)

		var cancelReqMu sync.Mutex
		var cancelReqFn func()
		const cancelIdx = 2
		var wg sync.WaitGroup
		wg.Add(maxRequests)

		for i := 0; i < maxRequests; i++ {
			go func(idx int) {
				defer wg.Done()
				rctx := ctx
				if idx == cancelIdx {
					cancelReqMu.Lock()
					rctx, cancelReqFn = context.WithCancel(ctx)
					cancelReqMu.Unlock()
				}
				_, e := limiter.RoundTrip(req.WithContext(rctx))
				if e != nil && idx != cancelIdx {
					stHelper.Must(e)
					return
				}

				if idx == cancelIdx {
					if !errors.Is(e, context.Canceled) {
						st.Errorf("expected context.Canceled error, got %v", e)
					}
				} else if e != nil {
					st.Errorf("unexpected error for request %d: %v", idx, e)
				}
			}(i)
		}

		// wait for goroutines to start
		time.Sleep(50 * time.Millisecond)

		cancelReqMu.Lock()
		cancelReqFn()
		cancelReqMu.Unlock()

		// wait for goroutines to finish
		wg.Wait()

		if count := reqCounter.Load(); count != maxRequests-1 {
			st.Errorf("expected %d requests, got %d", maxRequests-1, count)
		}
	})
}

func TestLimiter_Fixed(t *testing.T) {
	helper := test.New(t)
	logger, _ := test.NewLogger()

	ctx, cancelFn := context.WithCancel(context.Background())
	t.Cleanup(cancelFn)

	const period = "2s"

	limits, err := throttle.ConfigureThrottles(ctx, config.Throttles{
		&config.Throttle{
			Mode:         "wait",
			Period:       period,
			PerPeriod:    1,
			PeriodWindow: "fixed",
		},
	}, logger.WithContext(ctx))
	helper.Must(err)

	backend := &http.Transport{
		MaxConnsPerHost: 1,
	}

	limiter := throttle.NewLimiter(backend, nil)
	if limiter != nil {
		t.Errorf("expected nil Limiter, got %v", limiter)
	}

	limiter = throttle.NewLimiter(backend, limits)
	if limiter == nil {
		t.Errorf("expected configured Limiter, got %v", limiter)
	}

	t.Run("successful requests", func(st *testing.T) {
		const maxRequests = 4
		periodDuration, _ := time.ParseDuration(period)
		var expectedDuration = periodDuration * maxRequests // first req resets the period too

		stHelper := test.New(st)

		reqCounter := &atomic.Int32{}
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			reqCounter.Add(1)
		}))
		defer origin.Close()
		rctx, tCancel := context.WithCancel(ctx)
		defer tCancel()
		st.Cleanup(func() {
			tCancel()
			origin.Close()
		})

		req, rerr := http.NewRequest(http.MethodGet, origin.URL, nil)
		stHelper.Must(rerr)

		startTime := time.Now()
		wg := sync.WaitGroup{}
		wg.Add(maxRequests)
		for i := 0; i < maxRequests; i++ {
			go func() {
				defer wg.Done()
				resp, e := limiter.RoundTrip(req.WithContext(rctx))
				if e != nil {
					st.Errorf("unexpected error: %v", e)
					return
				}
				if resp.StatusCode != http.StatusOK {
					st.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
				}
			}()
			time.Sleep(periodDuration)
		}
		wg.Wait()
		duration := time.Since(startTime)

		if reqCounter.Load() != maxRequests {
			st.Errorf("expected %d requests, got %d", maxRequests, reqCounter.Load())
		}

		if !fuzzyEqual(duration, expectedDuration, time.Millisecond*100) {
			st.Errorf("expected duration around %v, got %v", expectedDuration, duration)
		}
		st.Logf("duration: %v, expected: %v", duration, expectedDuration)
	})
}

func TestLimiter_Block(t *testing.T) {
	helper := test.New(t)
	logger, _ := test.NewLogger()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	const period = "2s"

	limits, err := throttle.ConfigureThrottles(ctx, config.Throttles{
		&config.Throttle{
			Mode:         "block",
			Period:       period,
			PerPeriod:    1,
			PeriodWindow: "sliding",
		},
	}, logger.WithContext(ctx))
	helper.Must(err)

	backend := &http.Transport{
		MaxConnsPerHost: 1,
	}

	limiter := throttle.NewLimiter(backend, limits)
	if limiter == nil {
		t.Errorf("expected configured Limiter, got %v", limiter)
	}

	t.Run("successful and blocked requests", func(st *testing.T) {
		const maxRequests = 4
		stHelper := test.New(st)

		reqCounter := &atomic.Int32{}
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			reqCounter.Add(1)
		}))
		rctx, tCancel := context.WithCancel(ctx)
		st.Cleanup(func() {
			tCancel()
			origin.Close()
		})

		req, rerr := http.NewRequest(http.MethodGet, origin.URL, nil)
		stHelper.Must(rerr)

		startTime := time.Now()
		wg := sync.WaitGroup{}
		wg.Add(maxRequests)
		rateLimitErrs := &atomic.Uint32{}
		for i := 0; i < maxRequests; i++ {
			go func(idx int) {
				defer wg.Done()
				_, e := limiter.RoundTrip(req.WithContext(rctx))
				if errors.Is(e, couperErrors.BackendThrottleExceeded) {
					rateLimitErrs.Add(1)
				}
			}(i)
		}
		wg.Wait()
		st.Logf("duration: %v", time.Since(startTime))

		if reqCounter.Load() != 1 {
			st.Errorf("expected 1 request, got %d", reqCounter.Load())
		}

		if rateLimitErrs.Load() != 3 {
			st.Errorf("expected 3 rate limit errors, got %d", rateLimitErrs.Load())
		}
	})
}

func fuzzyEqual(a, b, fuzz time.Duration) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= fuzz
}
