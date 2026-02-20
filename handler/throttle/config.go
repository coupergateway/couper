package throttle

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/coupergateway/couper/config"
	"github.com/sirupsen/logrus"
)

const (
	notSet = iota
	modeBlock
	modeWait
	windowFixed
	windowSliding
)

// Throttle represents a throttle configuration.
type Throttle struct {
	count       *atomic.Uint64
	logger      *logrus.Entry
	mode        int
	perPeriod   uint64
	period      time.Duration
	periodStart time.Time
	quitCh      <-chan struct{}
	ringBuffer  *ringBuffer
	window      int
}

type Throttles []*Throttle

func ConfigureThrottles(ctx context.Context, limits config.Throttles, logger *logrus.Entry) (Throttles, error) {
	var (
		mode      int
		throttles Throttles
		window    int
	)

	uniqueDurations := make(map[time.Duration]struct{})

	for _, limit := range limits {
		d, err := config.ParseDuration("period", limit.Period, 0)
		if err != nil {
			return nil, err
		}

		if d == 0 {
			return nil, fmt.Errorf("'period' must not be 0 (zero)")
		}
		if limit.PerPeriod == 0 {
			return nil, fmt.Errorf("'per_period' must not be 0 (zero)")
		}

		if _, ok := uniqueDurations[time.Duration(d.Nanoseconds())]; ok {
			return nil, fmt.Errorf("duplicate period (%q) found", limit.Period)
		}

		uniqueDurations[time.Duration(d.Nanoseconds())] = struct{}{}

		switch limit.PeriodWindow {
		case "":
			fallthrough
		case "sliding":
			window = windowSliding
		case "fixed":
			window = windowFixed
		default:
			return nil, fmt.Errorf("unsupported 'period_window' (%q) given", limit.PeriodWindow)
		}

		switch limit.Mode {
		case "":
			fallthrough
		case "wait":
			mode = modeWait
		case "block":
			mode = modeBlock
		default:
			return nil, fmt.Errorf("unsupported 'mode' (%q) given", limit.Mode)
		}

		t := &Throttle{
			count:     &atomic.Uint64{},
			logger:    logger,
			mode:      mode,
			perPeriod: limit.PerPeriod,
			period:    time.Duration(d.Nanoseconds()),
			quitCh:    ctx.Done(),
			window:    window,
		}

		if t.window == windowSliding {
			t.ringBuffer = newRingBuffer(t.perPeriod)
		}

		throttles = append(throttles, t)
	}

	// Sort 'throttles' by 'period' DESC.
	sort.Slice(throttles, func(i, j int) bool {
		return throttles[i].period > throttles[j].period
	})

	return throttles, nil
}
