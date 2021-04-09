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
	println(`
Couper usage:
  couper <cmd> <options>

Available commands:
  run		Start the server with given configuration file.
  version	Print the current version and build information.
  help		Usage for given command.

Examples:
  couper run
  couper run -f couper.hcl
  couper run -watch -log-format json -log-pretty -p 3000
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
	if len(args) == 0 {
		h.Usage()
		return fmt.Errorf("missing command argument")
	}
	cmd := NewCommand(h.ctx, args[0])
	if cmd == nil {
		return fmt.Errorf("unknown command: %s", args[0])
	}
	cmd.Usage()
	return nil
}

func (h Help) Usage() {
	println("Usage of help:\n  help <command>	Print usage information of given command.")
}
