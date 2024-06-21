package ratelimit

import (
	"sync"
	"time"
)

type ringBuffer struct {
	buf []time.Time
	len uint64
	mu  sync.RWMutex
	r   uint64
	w   uint64
}

// newRingBuffer creates a new ringBuffer
// instance. ringBuffer is thread safe.
func newRingBuffer(len uint64) *ringBuffer {
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
