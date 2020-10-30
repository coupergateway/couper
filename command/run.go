package command

import (
	"context"
	"flag"

	"github.com/avenga/couper/config/env"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/server"
	"github.com/sirupsen/logrus"
)

var _ Cmd = &Run{}

// Run starts the frontend gateway server and listen
// for requests on the configured hosts and ports.
type Run struct {
	context context.Context
}

func NewRun(ctx context.Context) *Run {
	return &Run{context: ctx}
}

func (r Run) Execute(args Args, config *config.Gateway, logEntry *logrus.Entry) error {
	httpConf := runtime.NewHTTPConfig(config)

	// TODO: Extract and execute flagSet & env handling in a more generic way for future commands.
	set := flag.NewFlagSet("settings", flag.ContinueOnError)
	set.StringVar(&httpConf.HealthPath, "health-path", httpConf.HealthPath, "-health-path /healthz")
	set.IntVar(&httpConf.ListenPort, "p", httpConf.ListenPort, "-p 8080")
	set.BoolVar(&httpConf.UseXFH, "xfh", httpConf.UseXFH, "-xfh")
	set.StringVar(&httpConf.RequestIDFormat, "request-id-format", httpConf.RequestIDFormat, "-request-id-format uuid4")
	if err := set.Parse(args.Filter(set)); err != nil {
		return err
	}
	envConf := &runtime.HTTPConfig{}
	env.Decode(envConf)
	httpConf = httpConf.Merge(envConf)

	// logEntry has still the 'daemon' type which can be used for config related load errors.
	srvMux := runtime.NewServerConfiguration(config, httpConf, logEntry)

	serverList, listenCmdShutdown := server.NewServerList(r.context, logEntry.Logger, httpConf, srvMux)
	for _, srv := range serverList {
		srv.Listen()
	}
	listenCmdShutdown()
	return nil
}

func (r Run) Usage() string {
	panic("implement me")
}
