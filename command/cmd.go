package command

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config"
)

type Cmd interface {
	Execute(args Args, config *config.Couper, logger *logrus.Entry) error
	Usage()
}

func NewCommand(ctx context.Context, cmd string) Cmd {
	switch strings.ToLower(cmd) {
	case "run":
		return NewRun(ContextWithSignal(ctx))
	case "help":
		return NewHelp(ctx)
	case "version":
		return NewVersion()
	case "verify":
		return NewVerify()
	default:
		return nil
	}
}
