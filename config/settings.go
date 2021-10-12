package config

import (
	"flag"
	"fmt"
	"strings"
)

type AcceptForwarded struct {
	port, protocol, host bool
}

func (a *AcceptForwarded) Set(forwarded []string) error {
	a.protocol = false
	a.host = false
	a.port = false
	for _, part := range forwarded {
		switch strings.TrimSpace(part) {
		case "":
			continue
		case "port":
			a.port = true
		case "proto":
			a.protocol = true
		case "host":
			a.host = true
		default:
			return fmt.Errorf("invalid X-Forwarded-* name (%s)", part)
		}
	}

	return nil
}

func (a AcceptForwarded) String() string {
	var parts []string

	if a.protocol {
		parts = append(parts, "proto")
	}
	if a.host {
		parts = append(parts, "host")
	}
	if a.port {
		parts = append(parts, "port")
	}

	return strings.Join(parts, ",")
}

const otelCollectorEndpoint = "localhost:4317"

// DefaultSettings defines the <DefaultSettings> object.
var DefaultSettings = Settings{
	DefaultPort:              8080,
	HealthPath:               "/healthz",
	LogFormat:                "common",
	LogLevel:                 "info",
	LogPretty:                false,
	NoProxyFromEnv:           false,
	RequestIDBackendHeader:   "Couper-Request-ID",
	RequestIDClientHeader:    "Couper-Request-ID",
	RequestIDFormat:          "common",
	TelemetryMetricsEndpoint: otelCollectorEndpoint,
	TelemetryMetricsExporter: "prometheus",
	TelemetryMetricsPort:     9090, // default prometheus port
	TelemetryServiceName:     "couper",
	TelemetryTracesEndpoint:  otelCollectorEndpoint,
	XForwardedHost:           false,

	// TODO: refactor
	AcceptForwardedURL: []string{},
	AcceptForwarded:    &AcceptForwarded{},
}

// Settings represents the <Settings> object.
type Settings struct {
	AcceptForwarded *AcceptForwarded

	AcceptForwardedURL        []string `hcl:"accept_forwarded_url,optional"`
	DefaultPort               int      `hcl:"default_port,optional"`
	HealthPath                string   `hcl:"health_path,optional"`
	LogFormat                 string   `hcl:"log_format,optional"`
	LogLevel                  string   `hcl:"log_level,optional"`
	LogPretty                 bool     `hcl:"log_pretty,optional"`
	NoProxyFromEnv            bool     `hcl:"no_proxy_from_env,optional"`
	RequestIDAcceptFromHeader string   `hcl:"request_id_accept_from_header,optional"`
	RequestIDBackendHeader    string   `hcl:"request_id_backend_header,optional"`
	RequestIDClientHeader     string   `hcl:"request_id_client_header,optional"`
	RequestIDFormat           string   `hcl:"request_id_format,optional"`
	SecureCookies             string   `hcl:"secure_cookies,optional"`
	TLSDevProxy               List     `hcl:"https_dev_proxy,optional"`
	TelemetryMetrics          bool     `hcl:"beta_metrics,optional"`
	TelemetryMetricsEndpoint  string   `hcl:"beta_metrics_endpoint,optional"`
	TelemetryMetricsExporter  string   `hcl:"beta_metrics_exporter,optional"`
	TelemetryMetricsPort      int      `hcl:"beta_metrics_port,optional"`
	TelemetryServiceName      string   `hcl:"beta_service_name,optional"`
	TelemetryTraces           bool     `hcl:"beta_traces,optional"`
	TelemetryTracesEndpoint   string   `hcl:"beta_traces_endpoint,optional"`
	XForwardedHost            bool     `hcl:"xfh,optional"`
}

var _ flag.Value = &List{}

type List []string

func (s *List) String() string {
	return strings.Join(*s, ",")
}

func (s *List) Set(val string) error {
	if len(*s) > 0 { // argument priority over settings
		*s = nil
	}
	*s = append(*s, strings.Split(val, ",")...)
	return nil
}

func (s *Settings) SetAcceptForwarded() error {
	return s.AcceptForwarded.Set(s.AcceptForwardedURL)
}

func (s *Settings) AcceptsForwardedPort() bool {
	return s.AcceptForwarded.port
}

func (s *Settings) AcceptsForwardedProtocol() bool {
	return s.AcceptForwarded.protocol
}

func (s *Settings) AcceptsForwardedHost() bool {
	return s.AcceptForwarded.host
}
