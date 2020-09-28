package runtime

import (
	"flag"
	"os"
	"path/filepath"
	"time"

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
	ListenPort int    `env:"port"`
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
	HealthPath: "/healthz",
	Timings: HTTPTimings{
		IdleTimeout:       time.Second * 60,
		ReadHeaderTimeout: time.Second * 10,
		ShutdownDelay:     time.Second * 5,
		ShutdownTimeout:   time.Second * 5,
	},
	ListenPort: 8080,
}

var (
	flagHealthPath = flag.String("health-path", DefaultConfig.HealthPath, "-health-path /healthz")
	flagLogFormat  = flag.String("log-format", "default", "-log-format json")
	flagPort       = flag.Int("p", DefaultConfig.ListenPort, "-p 8080")
	flagXFH        = flag.Bool("xfh", DefaultConfig.UseXFH, "-xfh")
)

func NewHTTPConfig() *HTTPConfig {
	if !flag.Parsed() {
		flag.Parse()
	}

	conf := *DefaultConfig
	conf.HealthPath = *flagHealthPath
	conf.ListenPort = *flagPort
	conf.LogFormat = *flagLogFormat
	conf.UseXFH = *flagXFH

	env.Decode(&conf)
	return &conf
}

func SetWorkingDirectory(configFile string) error {
	return os.Chdir(filepath.Dir(configFile))
}
