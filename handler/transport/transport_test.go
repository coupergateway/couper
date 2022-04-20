package transport

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func Test_parseDuration(t *testing.T) {
	log := logrus.New()

	var target time.Duration
	parseDuration("1ms", &target, log.WithContext(context.TODO()))

	if target != 1000000 {
		t.Errorf("Unexpected duration given: %#v", target)
	}
}
