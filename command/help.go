package command

import (
	"context"
	"fmt"

	"github.com/avenga/couper/config"
	"github.com/sirupsen/logrus"
)

var _ Cmd = &Help{}

// Synopsis shows available commands and options.
func Synopsis() { // TODO: generate from command and options list
	println(`Couper usage:

  couper <cmd> <options>

Available commands:

  run		Start the server with given configuration file.
  help		Usage for given command.
  version	Print the current version and build information.
`)
}

type Help struct {
	ctx context.Context
}

func NewHelp(ctx context.Context) *Help {
	return &Help{ctx: ctx}
}

func (h Help) Execute(args Args, _ *config.Couper, _ *logrus.Entry) error {
	if len(args) == 0 {
		h.Usage()
		return fmt.Errorf("missing command argument")
	}
	NewCommand(h.ctx, args[0]).Usage()
	return nil
}

func (h Help) Usage() {
	println("Usage of help:\n  help <command>	Print usage information of given command.")
	Synopsis()
}
