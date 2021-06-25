package test

import (
	"testing"

	"github.com/avenga/couper/errors"
)

type Helper struct {
	tb testing.TB
}

func New(tb testing.TB) *Helper {
	return &Helper{tb}
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
