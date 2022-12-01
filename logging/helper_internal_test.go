package logging

import (
	"testing"
	"time"
)

func TestHelper_roundMS(t *testing.T) {
	type testCase struct {
		dur time.Duration
		exp float64
	}

	for _, tc := range []testCase{
		{1234567 * time.Nanosecond, 1.2350},
		{123456 * time.Microsecond, 123.456},
		{123456 * time.Microsecond, 123.456000},
		{123 * time.Millisecond, 123.0},
	} {
		if got := RoundMS(tc.dur); got != tc.exp {
			t.Errorf("expected '%#v', got '%#v'", tc.exp, got)
		}
	}
}
