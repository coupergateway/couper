package runtime

import (
	"os"
	"path/filepath"
	"time"

	"github.com/avenga/couper/config"
)

type Port string

type HostHandlers map[string]*ServerMux

type EntrypointHandlers map[Port]HostHandlers

// ServerMux represents the ServerMux struct.
type ServerMux struct {
	Server *config.Server
	Mux    *Mux
}

// HTTPConfig represents the configuration of the ingress HTTP server.
type HTTPConfig struct {
	HealthPath      string `env:"health_path"`
	ListenPort      int    `env:"default_port"`
	UseXFH          bool   `env:"xfh"`
	RequestIDFormat string `env:"request_id_format"`
	Timings         HTTPTimings
}

type HTTPTimings struct {
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	// ShutdownDelay determines the time between marking the http server
	// as unhealthy and calling the final shutdown method which denies accepting new requests.
	ShutdownDelay time.Duration
	// ShutdownTimeout is the context duration for shutting down the http server. Running requests
	// gets answered and those which exceeded this timeout getting lost. In combination with
	// ShutdownDelay the load-balancer should have picked another instance already.
	ShutdownTimeout time.Duration
}

// DefaultConfig sets some defaults for runtime.
var DefaultConfig = &Config{
	File:      "couper.hcl",
	LogFormat: "common",
}

// DefaultHTTP sets some defaults for the ingress HTTP server.
var DefaultHTTP = &HTTPConfig{
	HealthPath: "/healthz",
	Timings: HTTPTimings{
		IdleTimeout:       time.Second * 60,
		ReadHeaderTimeout: time.Second * 10,
		ShutdownDelay:     time.Second * 5,
		ShutdownTimeout:   time.Second * 5,
	},
	ListenPort:      8080,
	RequestIDFormat: "common",
}

// NewHTTPConfig creates the server config which could be overridden in order:
// internal.defaults -> config.settings -> flag.args -> env.vars
func NewHTTPConfig(c *config.Gateway) *HTTPConfig {
	defaultConf := *DefaultHTTP
	conf := &defaultConf
	if c != nil && c.Settings != nil {
		conf.Merge(newHTTPConfigFrom(c.Settings))
	}

	return conf
}

func newHTTPConfigFrom(s *config.Settings) *HTTPConfig {
	return &HTTPConfig{
		HealthPath:      s.HealthPath,
		ListenPort:      s.DefaultPort,
		UseXFH:          s.XForwardedHost,
		RequestIDFormat: s.RequestIDFormat,
		Timings:         DefaultHTTP.Timings,
	}
}

func (c *HTTPConfig) Merge(o *HTTPConfig) *HTTPConfig {
	if o == nil {
		return c
	}

	if o.HealthPath != "" {
		c.HealthPath = o.HealthPath
	}

	if o.ListenPort != 0 {
		c.ListenPort = o.ListenPort
	}

	if o.UseXFH != c.UseXFH {
		c.UseXFH = o.UseXFH
	}

	if o.RequestIDFormat != "" {
		c.RequestIDFormat = o.RequestIDFormat
	}

	return c
}

func SetWorkingDirectory(configFile string) (string, error) {
	if err := os.Chdir(filepath.Dir(configFile)); err != nil {
		return "", err
	}
	return os.Getwd()
}
