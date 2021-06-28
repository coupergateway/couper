package config

import (
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

// DefaultSettings defines the <DefaultSettings> object.
var DefaultSettings = Settings{
	DefaultPort:        8080,
	HealthPath:         "/healthz",
	LogFormat:          "common",
	NoProxyFromEnv:     false,
	RequestIDFormat:    "common",
	XForwardedHost:     false,
	AcceptForwarded:    &AcceptForwarded{},
	AcceptForwardedURL: []string{},
}

// Settings represents the <Settings> object.
type Settings struct {
	DefaultPort        int      `hcl:"default_port,optional"`
	HealthPath         string   `hcl:"health_path,optional"`
	LogFormat          string   `hcl:"log_format,optional"`
	LogPretty          bool     `hcl:"log_pretty,optional"`
	NoProxyFromEnv     bool     `hcl:"no_proxy_from_env,optional"`
	RequestIDFormat    string   `hcl:"request_id_format,optional"`
	SecureCookies      string   `hcl:"secure_cookies,optional"`
	XForwardedHost     bool     `hcl:"xfh,optional"`
	AcceptForwardedURL []string `hcl:"accept_forwarded_url,optional"`
	AcceptForwarded    *AcceptForwarded
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
