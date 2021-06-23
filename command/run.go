package command

import (
	"context"
	"flag"
	"fmt"
	"strings"
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
	set.Var(&AcceptForwardedValue{s: &settings}, "accept-forwarded-url", "-accept-forwarded-url [proto][,host][,port]")
	set.BoolVar(&settings.NoProxyFromEnv, "no-proxy-from-env", settings.NoProxyFromEnv, "-no-proxy-from-env")
	set.StringVar(&settings.RequestIDFormat, "request-id-format", settings.RequestIDFormat, "-request-id-format uuid4")
	set.StringVar(&settings.SecureCookies, "secure-cookies", settings.SecureCookies, "-secure-cookies strip")
	return &Run{
		context:  ctx,
		flagSet:  set,
		settings: &settings,
	}
}

type AcceptForwardedValue struct {
	s *config.Settings
}

func (a AcceptForwardedValue) String() string {
	if a.s == nil || a.s.AcceptForwarded == nil {
		return ""
	}
	return a.s.AcceptForwarded.String()
}

func (a AcceptForwardedValue) Set(s string) error {
	forwarded := strings.Split(s, ",")
	err := a.s.AcceptForwarded.Set(forwarded)
	if err != nil {
		return err
	}
	a.s.AcceptForwardedURL = forwarded
	return nil
}

func (r *Run) Execute(args Args, config *config.Couper, logEntry *logrus.Entry) error {
	r.settingsMu.Lock()
	*r.settings = *config.Settings
	r.settingsMu.Unlock()

	if flag := r.flagSet.Lookup("accept-forwarded-url"); flag != nil {
		if afv, ok := flag.Value.(*AcceptForwardedValue); ok {
			afv.s = r.settings
		}
	}

	if err := r.flagSet.Parse(args.Filter(r.flagSet)); err != nil {
		return err
	}

	if config.Settings.SecureCookies != "" &&
		config.Settings.SecureCookies != writer.SecureCookiesStrip {
		return fmt.Errorf("invalid value for the -secure-cookies flag given: '%s' only 'strip' is supported", config.Settings.SecureCookies)
	}

	// Some remapping due to flag set pre-definition
	env.Decode(r.settings)
	err := r.settings.SetAcceptForwarded()
	if err != nil {
		return err
	}
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
