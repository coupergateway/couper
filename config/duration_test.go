package config_test

import (
	"testing"
	"time"

	"github.com/avenga/couper/config"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		duration string
		_default time.Duration
		want     time.Duration
		err      string
	}{
		{"1s", time.Hour, time.Second, ""},
		{"0m", time.Hour, 0, ""},
		{"1h1s1m", time.Hour, 3661 * time.Second, ""},
		{"", time.Hour, time.Hour, ""},
		{"invalid", time.Hour, 0, `my-duration: time: invalid duration "invalid"`},
		{"1sec", time.Hour, 0, `my-duration: time: unknown unit "sec" in duration "1sec"`},
		{"-3s", time.Hour, 0, `my-duration: cannot be negative: '-3s'`},
	}
	for _, tt := range tests {
		t.Run("", func(subT *testing.T) {
			duration, err := config.ParseDuration("my-duration", tt.duration, tt._default)
			if duration != tt.want {
				subT.Errorf("unexpected duration, want: %q, got: %q", tt.want, duration)
			}
			if err != nil && err.Error() != tt.err {
				subT.Errorf("unexpected error,\n\twant: %q\n\tgot:  %q", tt.err, err.Error())
			}
			if err == nil && tt.err != "" {
				subT.Errorf("expected error %q, got: %v", tt.err, nil)
			}
		})
	}
}
