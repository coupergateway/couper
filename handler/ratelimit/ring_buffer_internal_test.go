package ratelimit

import (
	"testing"
	"time"
)

func TestRingBuffer_New(t *testing.T) {
	r := newRingBuffer(3)

	if len(r.buf) != 3 {
		t.Errorf("Unexpected r.buf: %#v", r.buf)
	}
	if r.len != 3 {
		t.Errorf("Unexpected r.len: %d", r.len)
	}
	if r.r != 0 {
		t.Errorf("Unexpected r.r: %d", r.r)
	}
	if r.w != 2 {
		t.Errorf("Unexpected r.w: %d", r.w)
	}
}

func TestRingBuffer_Put(t *testing.T) {
	now := time.Now()

	r := newRingBuffer(3)

	if !r.buf[0].IsZero() {
		t.Errorf("Unexpected r.buf[0]: %#v", r.buf[0])
	}
	if !r.buf[1].IsZero() {
		t.Errorf("Unexpected r.buf[1]: %#v", r.buf[1])
	}
	if !r.buf[2].IsZero() {
		t.Errorf("Unexpected r.buf[2]: %#v", r.buf[2])
	}

	r.put(now)

	if !r.buf[0].Equal(now) {
		t.Errorf("Unexpected r.buf[0]: %#v", r.buf[0])
	}
	if !r.buf[1].IsZero() {
		t.Errorf("Unexpected r.buf[1]: %#v", r.buf[1])
	}
	if !r.buf[2].IsZero() {
		t.Errorf("Unexpected r.buf[2]: %#v", r.buf[2])
	}

	if r.r != 1 {
		t.Errorf("Unexpected r.r: %d", r.r)
	}
	if r.w != 0 {
		t.Errorf("Unexpected r.w: %d", r.w)
	}

	r.put(now)

	if !r.buf[0].Equal(now) {
		t.Errorf("Unexpected r.buf[0]: %#v", r.buf[0])
	}
	if !r.buf[1].Equal(now) {
		t.Errorf("Unexpected r.buf[1]: %#v", r.buf[1])
	}
	if !r.buf[2].IsZero() {
		t.Errorf("Unexpected r.buf[2]: %#v", r.buf[2])
	}

	if r.r != 2 {
		t.Errorf("Unexpected r.r: %d", r.r)
	}
	if r.w != 1 {
		t.Errorf("Unexpected r.w: %d", r.w)
	}
}
