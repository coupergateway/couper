package test

import (
	"testing"
	"time"
)

//go:noinline
func parkGoroutine(release <-chan struct{}) {
	<-release
}

//go:noinline
func parkGoroutineViaWrapper(release <-chan struct{}) {
	parkGoroutine(release)
}

// The pprof goroutine profile groups goroutines by identical stack. Two
// goroutines in the same function reach it through different call paths, so
// they land in separate groups. NumGoroutines must sum all matching groups.
func TestNumGoroutines_SumsAcrossStackGroups(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	go parkGoroutine(release)
	go parkGoroutineViaWrapper(release)

	var n int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		n = NumGoroutines("test.parkGoroutine")
		if n == 2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("expected 2 goroutines in parkGoroutine, got: %d", n)
}
