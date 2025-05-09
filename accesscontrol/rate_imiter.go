package accesscontrol

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coupergateway/couper/accesscontrol/limiter"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

const (
	notSet = iota
	windowFixed
	windowSliding
)

var _ AccessControl = &RateLimiter{}

// RateLimiter represents an AC-RateLimiter object
type RateLimiter struct {
	name       string
	period     time.Duration
	perPeriod  int
	windowType int
	conf       *config.RateLimiter
	visitors   map[string]limiter.Limiter
	mu         sync.Mutex
}

// NewBasicAuth creates a new AC-RateLimiter object
func NewRateLimiter(name string, conf *config.RateLimiter) (*RateLimiter, error) {
	period, err := config.ParseDuration("period", conf.Period, 0)
	if err != nil {
		return nil, err
	}
	if period == 0 {
		return nil, fmt.Errorf("'period' must not be 0 (zero)")
	}

	if conf.PerPeriod == 0 {
		return nil, fmt.Errorf("'per_period' must not be 0 (zero)")
	}

	var windowType int
	switch conf.PeriodWindow {
	case "":
		fallthrough
	case "sliding":
		windowType = windowSliding
	case "fixed":
		windowType = windowFixed
	default:
		return nil, fmt.Errorf("unsupported 'period_window' (%q) given", conf.PeriodWindow)
	}

	rl := &RateLimiter{
		name:       name,
		period:     period,
		perPeriod:  conf.PerPeriod,
		windowType: windowType,
		conf:       conf,
		visitors:   make(map[string]limiter.Limiter),
	}

	return rl, nil
}

func (rl *RateLimiter) getVisitor(key string) limiter.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, exists := rl.visitors[key]
	if !exists {
		if rl.windowType == windowFixed {
			l = limiter.NewFixedWindowLimiter(rl.perPeriod, rl.period, time.Now)
		} else {
			l = limiter.NewSlidingWindowLimiter(rl.perPeriod, rl.period)
		}
		rl.visitors[key] = l
	}
	return l
}

// Validate implements the AccessControl interface
func (rl *RateLimiter) Validate(req *http.Request) error {
	ctx := eval.ContextFromRequest(req).HCLContext()
	keyVal, err := eval.ValueFromBodyAttribute(ctx, rl.conf.HCLBody(), "key")
	if err != nil {
		return errors.BetaRateLimiter.With(err)
	}

	keyValue := strings.TrimSpace(seetie.ValueToString(keyVal))
	if keyValue == "" {
		return errors.BetaRateLimiter.With(err).Message("Empty key value")
	}

	if !rl.getVisitor(keyValue).Allow() {
		return errors.BetaRateLimiter.Messagef("Request not allowed for %q", keyValue)
	}
	return nil
}
