package limiter

import (
	"fmt"
	"sync"
	"time"
)

type Limiter interface {
	Allow() bool
}

var _ Limiter = &FixedWindowLimiter{}
var _ Limiter = &SlidingWindowLimiter{}

type FixedWindowLimiter struct {
	mu       sync.Mutex
	window   time.Time
	count    int
	limit    int
	interval time.Duration
	clock    func() time.Time
}

func NewFixedWindowLimiter(limit int, interval time.Duration, clock func() time.Time) *FixedWindowLimiter {
	start := clock().Truncate(interval)
	return &FixedWindowLimiter{
		window:   start,
		limit:    limit,
		interval: interval,
		clock:    clock,
	}
}

func (l *FixedWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock()
	currentWindow := now.Truncate(l.interval)
	if currentWindow.After(l.window) {
		l.window = currentWindow
		l.count = 0
		fmt.Println("new window")
	}
	if l.count >= l.limit {
		return false
	}

	l.count++
	return true
}

type SlidingWindowLimiter struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int
	window   time.Duration
}

func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		requests: []time.Time{},
		limit:    limit,
		window:   window,
	}
}

func (l *SlidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for len(l.requests) > 0 && now.Sub(l.requests[0]) >= l.window {
		l.requests = l.requests[1:]
	}
	if len(l.requests) >= l.limit {
		return false
	}

	l.requests = append(l.requests, now)
	return true
}
