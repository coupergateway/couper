package transport

import (
	"testing"
	"time"
)

func Test_parseDuration(t *testing.T) {
	var target time.Duration
	parseDuration("1ms", &target)

	if target != 1000000 {
		t.Errorf("Unexpected duration given: %#v", target)
	}
}
