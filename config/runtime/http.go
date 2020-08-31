package runtime

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/config/env"
)

// Hosts represents the Hosts map.
type Hosts map[string]*ServerMux

// Ports represents the Ports map.
type Ports map[string]Hosts

// ServerMux represents the ServerMux struct.
type ServerMux struct {
	Server *config.Server
	Mux    *Mux
}

// HTTPConfig represents the configuration of the ingress HTTP server.
type HTTPConfig struct {
	ConfigFile string
	HCL        *config.Gateway
	ListenPort int `env:"port"`
	Lookups    Ports
	Timings    HTTPTimings
	UseXFH     bool `env:"xfh"`
	WorkDir    string
}

type HTTPTimings struct {
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// DefaultConfig sets some defaults for the ingress HTTP server.
var DefaultConfig = &HTTPConfig{
	ConfigFile: "couper.hcl",
	Timings: HTTPTimings{
		IdleTimeout:       time.Second * 60,
		ReadHeaderTimeout: time.Second * 10,
	},
	ListenPort: 8080,
}

// Configure configurates the ingress HTTP server.
func Configure(conf *HTTPConfig, logger *logrus.Entry) {
	if conf == nil {
		return
	}

	env.Decode(conf)

	if err := configureWorkDir(conf); err != nil {
		logger.Fatal(err)
	}

	if err := readHCL(conf); err != nil {
		logger.Fatal(err)
	}
}

func configureWorkDir(conf *HTTPConfig) error {
	err := os.Chdir(filepath.Dir(conf.ConfigFile))
	if err != nil {
		return err
	}

	conf.WorkDir, err = os.Getwd()
	if err != nil {
		return err
	}

	return nil
}

func readHCL(conf *HTTPConfig) error {
	if conf.ConfigFile == "" {
		return nil // For test cases
	}

	hcl, err := config.LoadFile(conf.ConfigFile)
	if err != nil {
		return err
	}

	conf.HCL = hcl

	return nil
}
