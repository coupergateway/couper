package runtime

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/config/env"
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
	ListenPort int    `env:"port"`
	LogFormat  string `env:"log_format"`
	UseXFH     bool   `env:"xfh"`
	Timings    HTTPTimings
}

type HTTPTimings struct {
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// DefaultConfig sets some defaults for the ingress HTTP server.
var DefaultConfig = &HTTPConfig{
	Timings: HTTPTimings{
		IdleTimeout:       time.Second * 60,
		ReadHeaderTimeout: time.Second * 10,
	},
	ListenPort: 8080,
}

var (
	flagPort      = flag.Int("p", DefaultConfig.ListenPort, "-p 8080")
	flagXFH       = flag.Bool("xfh", DefaultConfig.UseXFH, "-xfh")
	flagLogFormat = flag.String("log-format", "default", "-log-format json")
)

func NewHTTPConfig() *HTTPConfig {
	if !flag.Parsed() {
		flag.Parse()
	}

	conf := *DefaultConfig
	conf.UseXFH = *flagXFH
	conf.ListenPort = *flagPort
	conf.LogFormat = *flagLogFormat

	env.Decode(&conf)
	return &conf
}

func SetWorkingDirectory(configFile string) error {
	return os.Chdir(filepath.Dir(configFile))
}
