package accesscontrol

import (
	"net/http"
)

var _ AccessControl = &RateLimiter{}

// RateLimiter represents an AC-RateLimiter object
type RateLimiter struct {
	name string
}

// NewBasicAuth creates a new AC-RateLimiter object
func NewRateLimiter(name string) (*RateLimiter, error) {
	rl := &RateLimiter{
		name: name,
	}

	return rl, nil
}

// Validate implements the AccessControl interface
func (rl *RateLimiter) Validate(req *http.Request) error {
	// TODO implement
	return nil
}
