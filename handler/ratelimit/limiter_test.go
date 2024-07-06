package ratelimit_test

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
	"github.com/coupergateway/couper/handler/ratelimit"
	"github.com/coupergateway/couper/internal/test"
)

func TestNewLimiter(t *testing.T) {
	helper := test.New(t)
	logger, _ := test.NewLogger()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	const period = "2s"

	limits, err := ratelimit.ConfigureRateLimits(ctx, config.RateLimits{
		&config.RateLimit{
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

	limiter := ratelimit.NewLimiter(backend, nil)
	if limiter != nil {
		t.Errorf("expected nil Limiter, got %v", limiter)
	}

	limiter = ratelimit.NewLimiter(backend, limits)
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

		req, rerr := http.NewRequest(http.MethodGet, origin.URL, nil)
		stHelper.Must(rerr)

		rctx, tCancel := context.WithCancel(ctx)
		defer tCancel()

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

		if !fuzzyEqual(duration, expectedDuration, time.Millisecond*10) {
			st.Errorf("expected duration around %v, got %v", expectedDuration, duration)
		}
		st.Logf("duration: %v, expected: %v", duration, expectedDuration)
	})

	// Note: This test does not hit the coverage of the limiter until the limiter is using a buffered channel.
	t.Run("canceled request", func(st *testing.T) {
		const maxRequests = 4
		stHelper := test.New(st)

		reqCounter := &atomic.Int32{}
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			reqCounter.Add(1)
		}))
		defer origin.Close()

		req, rerr := http.NewRequest(http.MethodGet, origin.URL, nil)
		stHelper.Must(rerr)

		var cancelReqFn func()
		const cancelIdx = 2
		var wg sync.WaitGroup
		wg.Add(maxRequests)
		for i := 0; i < maxRequests; i++ {
			go func(idx int) {
				defer wg.Done()
				rctx := ctx
				if idx == cancelIdx {
					rctx, cancelReqFn = context.WithCancel(ctx)
				}
				_, e := limiter.RoundTrip(req.WithContext(rctx))
				if e != nil && idx != cancelIdx {
					stHelper.Must(e)
					return
				}
				if idx != cancelIdx {
					return
				}

				if !errors.Is(e, context.Canceled) {
					st.Errorf("expected context.Canceled error, got %v", err)
				}
			}(i)
		}

		// Cancel the request after some time so req-nr 3 is canceled while it should be in the queue.
		d, _ := time.ParseDuration(period)
		time.Sleep((cancelIdx + 1) * d)
		cancelReqFn()

		wg.Wait()

		if reqCounter.Load() != maxRequests-1 {
			st.Errorf("expected 2 requests, got %d", reqCounter.Load())
		}
	})
}

func fuzzyEqual(a, b, fuzz time.Duration) bool {
	return b <= a+fuzz && b >= a-fuzz
}
