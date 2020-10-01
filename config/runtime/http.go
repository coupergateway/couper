package runtime

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
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
	HealthPath string `env:"health_path"`
	ListenPort int    `env:"default_port"`
	LogFormat  string `env:"log_format"`
	UseXFH     bool   `env:"xfh"`
	Timings    HTTPTimings
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

// DefaultConfig sets some defaults for the ingress HTTP server.
var DefaultConfig = &HTTPConfig{
	LogFormat:  "common",
	HealthPath: "/healthz",
	Timings: HTTPTimings{
		IdleTimeout:       time.Second * 60,
		ReadHeaderTimeout: time.Second * 10,
		ShutdownDelay:     time.Second * 5,
		ShutdownTimeout:   time.Second * 5,
	},
	ListenPort: 8080,
}

// NewHTTPConfig creates the server config which could be overridden in order:
// internal.defaults -> config.settings -> flag.args -> env.vars
func NewHTTPConfig(c *config.Gateway, args command.Args) (*HTTPConfig, error) {
	defaultConf := *DefaultConfig
	conf := &defaultConf
	if c != nil && c.Settings != nil {
		conf.Merge(newHTTPConfigFrom(c.Settings))
	}

	set := flag.NewFlagSet("settings", flag.ContinueOnError)
	set.StringVar(&conf.HealthPath, "health-path", conf.HealthPath, "-health-path /healthz")
	set.StringVar(&conf.LogFormat, "log-format", conf.LogFormat, "-log-format json")
	set.IntVar(&conf.ListenPort, "p", conf.ListenPort, "-p 8080")
	set.BoolVar(&conf.UseXFH, "xfh", conf.UseXFH, "-xfh")
	if err := set.Parse(args.Filter(set)); err != nil {
		return nil, err
	}

	envConf := &HTTPConfig{}
	env.Decode(envConf)
	return conf.Merge(envConf), nil
}

func newHTTPConfigFrom(s *config.Settings) *HTTPConfig {
	return &HTTPConfig{
		HealthPath: s.HealthPath,
		ListenPort: s.DefaultPort,
		LogFormat:  s.LogFormat,
		UseXFH:     s.XForwardedHost,
		Timings:    DefaultConfig.Timings,
	}
}

func (c *HTTPConfig) Merge(o *HTTPConfig) *HTTPConfig {
	if o.HealthPath != "" {
		c.HealthPath = o.HealthPath
	}

	if o.ListenPort != 0 {
		c.ListenPort = o.ListenPort
	}

	if o.LogFormat != "" {
		c.LogFormat = o.LogFormat
	}

	if o.UseXFH != c.UseXFH {
		c.UseXFH = o.UseXFH
	}

	return c
}

func SetWorkingDirectory(configFile string) (string, error) {
	if err := os.Chdir(filepath.Dir(configFile)); err != nil {
		return "", err
	}
	return os.Getwd()
}
