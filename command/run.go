package command

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/env"
	"github.com/coupergateway/couper/config/runtime"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/server"
	"github.com/coupergateway/couper/server/writer"
	"github.com/coupergateway/couper/telemetry"
)

var _ Cmd = &Run{}
var RunCmdTestCallback func()
var RunCmdConfigTestCallback func(*config.Settings)

// Run starts the frontend gateway server and listen
// for requests on the configured hosts and ports.
type Run struct {
	context context.Context
	flagSet *flag.FlagSet
}

func NewRun(ctx context.Context) *Run {
	return &Run{
		context: ctx,
	}
}

// limitFn depends on current OS, set via build flags
var limitFn func(entry *logrus.Entry)

func (r *Run) Execute(args Args, config *config.Couper, logEntry *logrus.Entry) error {
	logEntry.WithField("files", config.Files.AsList()).Debug("loaded files")

	// apply command context
	config.Context = config.Context.(*eval.Context).WithContext(r.context)

	// apply cli flags to file settings obj
	r.flagSet = newFlagSet(config.Settings, "run")
	if err := r.flagSet.Parse(args.Filter(r.flagSet)); err != nil {
		return err
	}

	// TODO: move to config validation
	if config.Settings.SecureCookies != "" &&
		config.Settings.SecureCookies != writer.SecureCookiesStrip {
		return fmt.Errorf("invalid value for the -secure-cookies flag given: '%s' only 'strip' is supported", config.Settings.SecureCookies)
	}

	// finally apply environment variables to settings obj
	env.Decode(config.Settings)

	if err := config.Settings.ApplyAcceptForwarded(); err != nil {
		return err
	}

	if config.Settings.CAFile != "" {
		var err error
		config.Settings.Certificate, err = readCertificateFile(config.Settings.CAFile)
		if err != nil {
			return err
		}
		logEntry.Infof("configured with ca-certificate: %s", config.Settings.CAFile)
	}

	if RunCmdConfigTestCallback != nil {
		RunCmdConfigTestCallback(config.Settings)
	}

	timings := runtime.DefaultTimings
	env.Decode(&timings)

	memStore := cache.New(logEntry, r.context.Done())
	// logEntry has still the 'daemon' type which can be used for config related load errors.
	srvConf, err := runtime.NewServerConfiguration(config, logEntry, memStore)
	if err != nil {
		return err
	}
	errors.SetLogger(logEntry)

	err = telemetry.InitExporter(r.context, &telemetry.Options{
		MetricsCollectPeriod: time.Second * 2,
		Metrics:              config.Settings.TelemetryMetrics,
		MetricsEndpoint:      config.Settings.TelemetryMetricsEndpoint,
		MetricsExporter:      config.Settings.TelemetryMetricsExporter,
		MetricsPort:          config.Settings.TelemetryMetricsPort,
		ServiceName:          config.Settings.TelemetryServiceName,
		Traces:               config.Settings.TelemetryTraces,
		TracesEndpoint:       config.Settings.TelemetryTracesEndpoint,
	}, memStore, logEntry)
	if err != nil {
		return err
	}

	if limitFn != nil {
		limitFn(logEntry)
	}

	tlsDevPorts := make(server.TLSDevPorts)
	for _, ports := range config.Settings.TLSDevProxy {
		if err = tlsDevPorts.Add(ports); err != nil {
			return err
		}
	}

	servers, listenCmdShutdown, err := server.NewServers(r.context, config.Context, logEntry, config.Settings, &timings, srvConf)
	if err != nil {
		return err
	}
	var tlsServer []*http.Server

	for mappedListenPort := range tlsDevPorts {
		if _, exist := srvConf[mappedListenPort.Port()]; !exist {
			return errors.Configuration.Messagef("%s: target port not configured: %s", server.TLSProxyOption, mappedListenPort)
		}
	}

	for _, srv := range servers {
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

	if RunCmdTestCallback != nil {
		RunCmdTestCallback()
	}

	listenCmdShutdown()

	for _, s := range tlsServer {
		_ = s.Close()
		logEntry.Infof("Server closed: %s", s.Addr)
	}

	return nil
}

// readCertificateFile reads given file bytes and PEM decodes the certificates the
// same way x509.CertPool.AppendCertsFromPEM does.
// AppendCertsFromPEM method will be used on backend transport creation.
func readCertificateFile(file string) ([]byte, error) {
	cert, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("error reading ca-certificate: %v", err)
	} else if len(cert) == 0 {
		return nil, fmt.Errorf("error reading ca-certificate: empty file: %q", file)
	}

	hasValidCert := false
	pemCerts := cert[:]
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		certBytes := block.Bytes
		if _, err = x509.ParseCertificate(certBytes); err != nil {
			return nil, fmt.Errorf("error parsing pem ca-certificate: %q: %v", file, err)
		}

		hasValidCert = true
	}

	if !hasValidCert {
		return nil, fmt.Errorf("error parsing pem ca-certificate: has no valid X509 certificate")
	}

	return cert, nil
}

func (r *Run) Usage() {
	r.flagSet.Usage()
}

func newFlagSet(settings *config.Settings, cmdName string) *flag.FlagSet {
	set := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	set.StringVar(&settings.CAFile, "ca-file", settings.CAFile, "-ca-file certificate.pem")
	set.StringVar(&settings.HealthPath, "health-path", settings.HealthPath, "-health-path /healthz")
	set.IntVar(&settings.DefaultPort, "p", settings.DefaultPort, "-p 8080")
	set.BoolVar(&settings.XForwardedHost, "xfh", settings.XForwardedHost, "-xfh")
	set.Var(&settings.AcceptForwardedURL, "accept-forwarded-url", "-accept-forwarded-url [proto][,host][,port]")
	set.Var(&settings.TLSDevProxy, "https-dev-proxy", "-https-dev-proxy 8443:8080,9443:9000")
	set.BoolVar(&settings.NoProxyFromEnv, "no-proxy-from-env", settings.NoProxyFromEnv, "-no-proxy-from-env")
	set.StringVar(&settings.RequestIDAcceptFromHeader, "request-id-accept-from-header", settings.RequestIDAcceptFromHeader, "-request-id-accept-from-header X-UID")
	set.StringVar(&settings.RequestIDBackendHeader, "request-id-backend-header", settings.RequestIDBackendHeader, "-request-id-backend-header Couper-Request-ID")
	set.StringVar(&settings.RequestIDClientHeader, "request-id-client-header", settings.RequestIDClientHeader, "-request-id-client-header Couper-Request-ID")
	set.StringVar(&settings.RequestIDFormat, "request-id-format", settings.RequestIDFormat, "-request-id-format uuid4")
	set.StringVar(&settings.SecureCookies, "secure-cookies", settings.SecureCookies, "-secure-cookies strip")
	set.BoolVar(&settings.SendServerTimings, "send-server-timing-headers", settings.SendServerTimings, "-send-server-timing-headers")
	set.BoolVar(&settings.TelemetryMetrics, "beta-metrics", settings.TelemetryMetrics, "-beta-metrics")
	set.IntVar(&settings.TelemetryMetricsPort, "beta-metrics-port", settings.TelemetryMetricsPort, "-beta-metrics-port 9090")
	set.StringVar(&settings.TelemetryMetricsEndpoint, "beta-metrics-endpoint", settings.TelemetryMetricsEndpoint, "-beta-metrics-endpoint [host:port]")
	set.StringVar(&settings.TelemetryMetricsExporter, "beta-metrics-exporter", settings.TelemetryMetricsExporter, "-beta-metrics-exporter [name]")
	set.StringVar(&settings.TelemetryServiceName, "beta-service-name", settings.TelemetryServiceName, "-beta-service-name [name]")
	set.BoolVar(&settings.TelemetryTraces, "beta-traces", settings.TelemetryTraces, "-beta-traces")
	set.StringVar(&settings.TelemetryTracesEndpoint, "beta-traces-endpoint", settings.TelemetryTracesEndpoint, "-beta-traces-endpoint [host:port]")

	return set
}
