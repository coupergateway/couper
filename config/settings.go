package config

import (
	"fmt"
)

type AcceptForwarded struct {
	port, protocol, host bool
}

func (a *AcceptForwarded) Set(forwarded []string) error {
	for _, part := range forwarded {
		switch part {
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
	s := ""
	if a.protocol {
		s += "proto"
	}
	if a.host {
		if len(s) > 0 {
			s += ","
		}
		s += "host"
	}
	if a.port {
		if len(s) > 0 {
			s += ","
		}
		s += "port"
	}
	return s
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
	if (*s.AcceptForwarded == AcceptForwarded{}) && len(s.AcceptForwardedURL) > 0 {
		return s.AcceptForwarded.Set(s.AcceptForwardedURL)
	}

	return nil
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
