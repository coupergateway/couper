package config

import (
	"strings"
)

type AcceptForwarded struct {
	port, protocol, host bool
}

// DefaultSettings defines the <DefaultSettings> object.
var DefaultSettings = Settings{
	DefaultPort:        8080,
	HealthPath:         "/healthz",
	LogFormat:          "common",
	NoProxyFromEnv:     false,
	RequestIDFormat:    "common",
	XForwardedHost:     false,
	AcceptForwardedURL: "",
}

// Settings represents the <Settings> object.
type Settings struct {
	DefaultPort        int    `hcl:"default_port,optional"`
	HealthPath         string `hcl:"health_path,optional"`
	LogFormat          string `hcl:"log_format,optional"`
	LogPretty          bool   `hcl:"log_pretty,optional"`
	NoProxyFromEnv     bool   `hcl:"no_proxy_from_env,optional"`
	RequestIDFormat    string `hcl:"request_id_format,optional"`
	SecureCookies      string `hcl:"secure_cookies,optional"`
	XForwardedHost     bool   `hcl:"xfh,optional"`
	AcceptForwardedURL string `hcl:"accept_forwarded_url,optional"`
	AcceptForwarded    *AcceptForwarded
}

func (s *Settings) AcceptsForwardedPort() bool {
	return s.getAcceptForwarded().port
}

func (s *Settings) AcceptsForwardedProtocol() bool {
	return s.getAcceptForwarded().protocol
}

func (s *Settings) AcceptsForwardedHost() bool {
	return s.getAcceptForwarded().host
}

func (s *Settings) getAcceptForwarded() *AcceptForwarded {
	if s.AcceptForwarded == nil {
		s.AcceptForwarded = &AcceptForwarded{}
		parts := strings.Split(s.AcceptForwardedURL, ",")
		for _, part := range parts {
			switch part {
			case "port":
				s.AcceptForwarded.port = true
			case "proto":
				s.AcceptForwarded.protocol = true
			case "host":
				s.AcceptForwarded.host = true
			}
		}
	}
	return s.AcceptForwarded
}
