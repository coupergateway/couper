package command

import (
	"context"

	"github.com/coupergateway/couper/config"
	"github.com/sirupsen/logrus"
)

var _ Cmd = &Help{}

// Synopsis shows available commands and options.
func Synopsis() { // TODO: generate from command and options list
	println(`
Couper usage:
  couper <cmd> <options>

Available commands:
  help		Usage for given command.
  run		Start the server with given configuration file.
  verify	Verify the syntax of the given configuration file.
  version	Print the current version and build information.

Examples:
  couper run
  couper run -f couper.hcl
  couper run -watch -log-format json -log-pretty -p 3000
  couper verify -f couper.hcl
`)
}

type Help struct {
	ctx context.Context
}

func NewHelp(ctx context.Context) *Help {
	return &Help{ctx: ctx}
}

func (h Help) Execute(args Args, _ *config.Couper, _ *logrus.Entry) error {
	defer Synopsis()
	h.Usage()
	return nil
}

func (h Help) Usage() {
	println("Usage of help:\n  help <command>	Print usage information of given command.")
}
