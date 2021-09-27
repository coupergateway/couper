package test

import (
	"io"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/logging/hooks"
)

func NewLogger() (*logrus.Logger, *logrustest.Hook) {
	log := logrus.New()
	log.Out = io.Discard
	log.AddHook(&hooks.Error{})
	log.AddHook(&hooks.Context{})
	hook := logrustest.NewLocal(log)
	return log, hook
}
