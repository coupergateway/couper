package command

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/server"
	"github.com/avenga/couper/server/writer"
	"github.com/avenga/couper/telemetry"
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
	set.Var(&AcceptForwardedValue{settings: &settings}, "accept-forwarded-url", "-accept-forwarded-url [proto][,host][,port]")
	set.Var(&settings.TLSDevProxy, "https-dev-proxy", "-https-dev-proxy 8443:8080,9443:9000")
	set.BoolVar(&settings.NoProxyFromEnv, "no-proxy-from-env", settings.NoProxyFromEnv, "-no-proxy-from-env")
	set.StringVar(&settings.RequestIDAcceptFromHeader, "request-id-accept-from-header", settings.RequestIDAcceptFromHeader, "-request-id-accept-from-header X-UID")
	set.StringVar(&settings.RequestIDBackendHeader, "request-id-backend-header", settings.RequestIDBackendHeader, "-request-id-backend-header Couper-Request-ID")
	set.StringVar(&settings.RequestIDClientHeader, "request-id-client-header", settings.RequestIDClientHeader, "-request-id-client-header Couper-Request-ID")
	set.StringVar(&settings.RequestIDFormat, "request-id-format", settings.RequestIDFormat, "-request-id-format uuid4")
	set.StringVar(&settings.SecureCookies, "secure-cookies", settings.SecureCookies, "-secure-cookies strip")
	set.BoolVar(&settings.TelemetryMetrics, "beta-metrics", settings.TelemetryMetrics, "-beta-metrics")
	set.IntVar(&settings.TelemetryMetricsPort, "beta-metrics-port", settings.TelemetryMetricsPort, "-beta-metrics-port 9090")
	set.StringVar(&settings.TelemetryMetricsEndpoint, "beta-metrics-endpoint", settings.TelemetryMetricsEndpoint, "-beta-metrics-endpoint [host:port]")
	set.StringVar(&settings.TelemetryMetricsExporter, "beta-metrics-exporter", settings.TelemetryMetricsExporter, "-beta-metrics-exporter [name]")
	set.StringVar(&settings.TelemetryServiceName, "beta-service-name", settings.TelemetryServiceName, "-beta-service-name [name]")
	set.BoolVar(&settings.TelemetryTraces, "beta-traces", settings.TelemetryTraces, "-traces")
	set.StringVar(&settings.TelemetryTracesEndpoint, "beta-traces-endpoint", settings.TelemetryTracesEndpoint, "-beta-traces-endpoint [host:port]")
	return &Run{
		context:  ctx,
		flagSet:  set,
		settings: &settings,
	}
}

type AcceptForwardedValue struct {
	settings *config.Settings
}

func (a AcceptForwardedValue) String() string {
	if a.settings == nil || a.settings.AcceptForwarded == nil {
		return ""
	}
	return a.settings.AcceptForwarded.String()
}

func (a AcceptForwardedValue) Set(s string) error {
	forwarded := strings.Split(s, ",")
	err := a.settings.AcceptForwarded.Set(forwarded)
	if err != nil {
		return err
	}
	a.settings.AcceptForwardedURL = forwarded
	return nil
}

func (r *Run) Execute(args Args, config *config.Couper, logEntry *logrus.Entry) error {
	r.settingsMu.Lock()
	*r.settings = *config.Settings
	r.settingsMu.Unlock()

	if f := r.flagSet.Lookup("accept-forwarded-url"); f != nil {
		if afv, ok := f.Value.(*AcceptForwardedValue); ok {
			afv.settings = r.settings
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

	telemetry.InitExporter(r.context, &telemetry.Options{
		MetricsCollectPeriod: time.Second * 2,
		Metrics:              r.settings.TelemetryMetrics,
		MetricsEndpoint:      r.settings.TelemetryMetricsEndpoint,
		MetricsExporter:      r.settings.TelemetryMetricsExporter,
		MetricsPort:          r.settings.TelemetryMetricsPort,
		ServiceName:          r.settings.TelemetryServiceName,
		Traces:               r.settings.TelemetryTraces,
		TracesEndpoint:       r.settings.TelemetryTracesEndpoint,
	}, logEntry)

	// logEntry has still the 'daemon' type which can be used for config related load errors.
	srvConf, err := runtime.NewServerConfiguration(config, logEntry, cache.New(logEntry, r.context.Done()))
	if err != nil {
		return err
	}
	errors.SetLogger(logEntry)

	tlsDevPorts := make(server.TLSDevPorts)
	for _, ports := range config.Settings.TLSDevProxy {
		if err = tlsDevPorts.Add(ports); err != nil {
			return err
		}
	}

	serverList, listenCmdShutdown := server.NewServerList(r.context, config.Context, logEntry, config.Settings, &timings, srvConf)
	var tlsServer []*http.Server

	for mappedListenPort := range tlsDevPorts {
		if _, exist := srvConf[mappedListenPort.Port()]; !exist {
			return errors.Configuration.Messagef("%s: target port not configured: %s", server.TLSProxyOption, mappedListenPort)
		}
	}

	for _, srv := range serverList {
		if listenErr := srv.Listen(); listenErr != nil {
			return listenErr
		}

		_, port, splitErr := net.SplitHostPort(srv.Addr())
		if splitErr != nil {
			return splitErr
		}

		for _, tlsPort := range tlsDevPorts.Get(port) {
			tlsSrv, tlsErr := server.NewTLSProxy(srv.Addr(), tlsPort, logEntry, config.Settings)
			if tlsErr != nil {
				return tlsErr
			}
			tlsServer = append(tlsServer, tlsSrv)
			logEntry.Infof("couper is serving tls: %s -> %s", tlsPort, port)
		}
	}

	listenCmdShutdown()

	for _, s := range tlsServer {
		_ = s.Close()
		logEntry.Infof("Server closed: %s", s.Addr)
	}

	return nil
}

func (r *Run) Usage() {
	r.flagSet.Usage()
}
