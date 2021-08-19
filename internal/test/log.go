package test

import (
	"io"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/logging"
)

func NewLogger() (*logrus.Logger, *logrustest.Hook) {
	log := logrus.New()
	log.Out = io.Discard
	log.AddHook(&errors.LogHook{})
	log.AddHook(&logging.ContextHook{})
	hook := logrustest.NewLocal(log)
	return log, hook
}
