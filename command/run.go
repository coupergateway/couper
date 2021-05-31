package command

import (
	"context"
	"flag"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/server"
	"github.com/avenga/couper/server/writer"
)

var _ Cmd = &Run{}

// Run starts the frontend gateway server and listen
// for requests on the configured hosts and ports.
type Run struct {
	context  context.Context
	flagSet  *flag.FlagSet
	settings *config.Settings

	// required for testing purposes
	// TODO: provide a testable interface
	settingsMu sync.Mutex
}

func NewRun(ctx context.Context) *Run {
	settings := config.DefaultSettings
	set := flag.NewFlagSet("run", flag.ContinueOnError)
	set.StringVar(&settings.HealthPath, "health-path", settings.HealthPath, "-health-path /healthz")
	set.IntVar(&settings.DefaultPort, "p", settings.DefaultPort, "-p 8080")
	set.BoolVar(&settings.XForwardedHost, "xfh", settings.XForwardedHost, "-xfh")
	set.BoolVar(&settings.NoProxyFromEnv, "no-proxy-from-env", settings.NoProxyFromEnv, "-no-proxy-from-env")
	set.StringVar(&settings.RequestIDFormat, "request-id-format", settings.RequestIDFormat, "-request-id-format uuid4")
	set.StringVar(&settings.SecureCookies, "secure-cookies", settings.SecureCookies, "-secure-cookies strip")
	return &Run{
		context:  ctx,
		flagSet:  set,
		settings: &settings,
	}
}

func (r *Run) Execute(args Args, config *config.Couper, logEntry *logrus.Entry) error {
	r.settingsMu.Lock()
	*r.settings = *config.Settings
	r.settingsMu.Unlock()

	if err := r.flagSet.Parse(args.Filter(r.flagSet)); err != nil {
		return err
	}

	if config.Settings.SecureCookies != "" &&
		config.Settings.SecureCookies != writer.SecureCookiesStrip {
		return fmt.Errorf("invalid value for the -secure-cookies flag given: '%s' only 'strip' is supported", config.Settings.SecureCookies)
	}

	// Some remapping due to flag set pre-definition
	env.Decode(r.settings)
	r.settingsMu.Lock()
	config.Settings = r.settings
	r.settingsMu.Unlock()

	timings := runtime.DefaultTimings
	env.Decode(&timings)

	// logEntry has still the 'daemon' type which can be used for config related load errors.
	srvConf, err := runtime.NewServerConfiguration(config, logEntry, cache.New(logEntry, r.context.Done()))
	if err != nil {
		return err
	}
	errors.SetLogger(logEntry)

	serverList, listenCmdShutdown := server.NewServerList(r.context, config.Context, logEntry, config.Settings, &timings, srvConf)
	for _, srv := range serverList {
		if listenErr := srv.Listen(); listenErr != nil {
			return listenErr
		}
	}
	listenCmdShutdown()
	return nil
}

func (r *Run) Usage() {
	r.flagSet.Usage()
}
