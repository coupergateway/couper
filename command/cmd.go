package command

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
)

type Cmd interface {
	Execute(args Args, config *config.Gateway, logger *logrus.Entry) error
	Usage() string
}

func NewCommand(cmd string) Cmd {
	switch strings.ToLower(cmd) {
	case "run":
		return NewRun(ContextWithSignal(context.Background()))
	default:
		return nil
	}
}
