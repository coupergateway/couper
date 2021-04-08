package command

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
)

type Cmd interface {
	Execute(args Args, config *config.Couper, logger *logrus.Entry) error
	Usage() string
}

func NewCommand(ctx context.Context, cmd string) Cmd {
	switch strings.ToLower(cmd) {
	case "run":
		return NewRun(ContextWithSignal(ctx))
	case "version":
		return NewVersion()
	default:
		return nil
	}
}
