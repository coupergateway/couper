package test

import (
	"io"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/coupergateway/couper/logging/hooks"
)

func NewLogger() (*logrus.Logger, *logrustest.Hook) {
	log := logrus.New()
	log.Out = io.Discard
	log.AddHook(&hooks.Error{})
	log.AddHook(&hooks.Context{})
	log.AddHook(&hooks.CustomLogs{})
	hook := logrustest.NewLocal(log)
	return log, hook
}
