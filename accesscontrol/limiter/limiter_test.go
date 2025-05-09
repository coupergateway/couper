package limiter_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/coupergateway/couper/accesscontrol/limiter"
)

func Test_FixedWindowLimiter(t *testing.T) {
	// simulated time
	now := time.Date(2025, 5, 8, 12, 0, 0, 0, time.UTC)
	getTime := func() time.Time {
		return now
	}

	l := limiter.NewFixedWindowLimiter(3, 1*time.Second, getTime)
	// first 3 requests in the window are allowed
	for i := range 3 {
		if !l.Allow() {
			t.Errorf("Expected request %d to be allowed", i)
		}
	}
	// 4th request should be denied
	if l.Allow() {
		t.Error("Expected 4th request to be denied")
	}

	// advance time to next window
	now = now.Add(1 * time.Second)

	// new window, 3 requests allowed again
	for i := range 3 {
		if !l.Allow() {
			t.Errorf("Expected request %d (in new window) to be allowed", i)
		}
	}
	// again, 4th request in new window should be denied
	if l.Allow() {
		t.Error("Expected 4th request (in new window) to be denied")
	}
}

func Test_SlidingdWindowLimiter(t *testing.T) {
	l := limiter.NewSlidingWindowLimiter(5, 1*time.Second)
	results := []bool{}

	// trickle phase
	for range 5 {
		allow := l.Allow()
		results = append(results, allow)
		time.Sleep((200 * time.Millisecond))
	}

	// burst phase
	for range 5 {
		allow := l.Allow()
		results = append(results, allow)
		time.Sleep((10 * time.Millisecond))
	}

	// trickle phase
	for range 6 {
		allow := l.Allow()
		results = append(results, allow)
		time.Sleep((200 * time.Millisecond))
	}

	exp := []bool{
		true, true, true, true, true,
		true, false, false, false, false,
		false, true, true, true, true, true,
	}
	if cmp.Diff(exp, results) != "" {
		t.Errorf("Unexpected results:\n\tWant: %v\n\tGot:  %v", exp, results)
	}
}
