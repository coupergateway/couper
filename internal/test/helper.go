package test

import (
	"testing"

	"github.com/coupergateway/couper/errors"
)

type Helper struct {
	tb testing.TB
}

func New(tb testing.TB) *Helper {
	return &Helper{tb}
}

func (h *Helper) TestFailed() bool {
	h.tb.Helper()
	return h.tb.Failed()
}

func (h *Helper) Logf(msg string, args ...interface{}) {
	h.tb.Helper()
	h.tb.Logf(msg, args...)
}

func (h *Helper) Must(err error) {
	h.tb.Helper()
	if err != nil {
		if logErr, ok := err.(errors.GoError); ok {
			h.tb.Fatal(logErr.LogError())
			return
		}
		h.tb.Fatal(err)
	}
}
