package accesscontrol

import (
	"crypto/sha256"
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

type LimiterEntry struct {
	limiter  limiter.Limiter
	lastUsed time.Time
}

// RateLimiter represents an AC-RateLimiter object
type RateLimiter struct {
	name           string
	period         time.Duration
	perPeriod      int
	windowType     int
	conf           *config.RateLimiter
	limiterEntries map[[32]byte]*LimiterEntry
	mu             sync.Mutex
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
		name:           name,
		period:         period,
		perPeriod:      conf.PerPeriod,
		windowType:     windowType,
		conf:           conf,
		limiterEntries: make(map[[32]byte]*LimiterEntry),
	}
	rl.startLimiterGC()

	return rl, nil
}

func (rl *RateLimiter) getLimiter(key [32]byte) limiter.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, exists := rl.limiterEntries[key]
	if exists {
		l.lastUsed = time.Now()
		return l.limiter
	}

	var lim limiter.Limiter
	if rl.windowType == windowFixed {
		lim = limiter.NewFixedWindowLimiter(rl.perPeriod, rl.period, time.Now)
	} else {
		lim = limiter.NewSlidingWindowLimiter(rl.perPeriod, rl.period)
	}
	rl.limiterEntries[key] = &LimiterEntry{
		limiter:  lim,
		lastUsed: time.Now(),
	}
	return lim
}

// Validate implements the AccessControl interface
func (rl *RateLimiter) Validate(req *http.Request) error {
	ctx := eval.ContextFromRequest(req).HCLContext()
	keyVal, err := eval.ValueFromBodyAttribute(ctx, rl.conf.HCLBody(), "key")
	if err != nil {
		return errors.BetaRateLimiterKey.With(err)
	}

	keyValue := strings.TrimSpace(seetie.ValueToString(keyVal))
	if keyValue == "" {
		return errors.BetaRateLimiterKey.With(err).Message("Empty key value")
	}

	keyHash := sha256.Sum256([]byte(keyValue))
	if !rl.getLimiter(keyHash).Allow() {
		return errors.BetaRateLimiter.Messagef("Request not allowed for %q", keyValue)
	}
	return nil
}

func (rl *RateLimiter) startLimiterGC() {
	idleTimeout := 3 * rl.period
	interval := rl.period / 2
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for key, entry := range rl.limiterEntries {
				if now.Sub(entry.lastUsed) > idleTimeout {
					delete(rl.limiterEntries, key)
				}
			}
			rl.mu.Unlock()
		}
	}()
}
