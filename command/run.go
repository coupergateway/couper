package command

import (
	"context"
	"flag"
	"fmt"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
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

func (r Run) Execute(args Args, config *config.Couper, logEntry *logrus.Entry) error {
	// TODO: Extract and execute flagSet & env handling in a more generic way for future commands.
	set := flag.NewFlagSet("settings", flag.ContinueOnError)
	set.StringVar(&config.Settings.HealthPath, "health-path", config.Settings.HealthPath, "-health-path /healthz")
	set.IntVar(&config.Settings.DefaultPort, "p", config.Settings.DefaultPort, "-p 8080")
	set.BoolVar(&config.Settings.XForwardedHost, "xfh", config.Settings.XForwardedHost, "-xfh")
	set.BoolVar(&config.Settings.NoProxyFromEnv, "no-proxy-from-env", config.Settings.NoProxyFromEnv, "-no-proxy-from-env")
	set.StringVar(&config.Settings.RequestIDFormat, "request-id-format", config.Settings.RequestIDFormat, "-request-id-format uuid4")
	set.StringVar(&config.Settings.SecureCookies, "secure-cookies", config.Settings.SecureCookies, "-secure-cookies strip|enforce")
	if err := set.Parse(args.Filter(set)); err != nil {
		return err
	}

	if config.Settings.SecureCookies != "" &&
		config.Settings.SecureCookies != server.SecureCookiesStrip &&
		config.Settings.SecureCookies != server.SecureCookiesEnforce {
		return fmt.Errorf("Invalid value for the -secure-cookies flag given. Only 'strip' or 'enforce' is allowed.")
	}

	env.Decode(config.Settings)

	timings := runtime.DefaultTimings
	env.Decode(&timings)

	// logEntry has still the 'daemon' type which can be used for config related load errors.
	srvConf, err := runtime.NewServerConfiguration(config, logEntry, cache.New(logEntry, r.context.Done()))
	if err != nil {
		return err
	}

	serverList, listenCmdShutdown := server.NewServerList(r.context, config.Context, logEntry.Logger, config.Settings, &timings, srvConf)
	for _, srv := range serverList {
		srv.Listen()
	}
	listenCmdShutdown()
	return nil
}

func (r Run) Usage() string {
	panic("implement me")
}
