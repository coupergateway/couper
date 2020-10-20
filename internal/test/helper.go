package test

import "testing"

type Helper struct {
	tb testing.TB
}

func New(tb testing.TB) *Helper {
	return &Helper{tb}
}

func (h *Helper) Must(err error) {
	if err != nil {
		h.tb.Fatal(err)
	}
}
