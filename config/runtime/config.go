package runtime

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/config"
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

// Config represents the configuration of the ingress HTTP server.
type Config struct {
	ConfigFile        string
	HCL               *config.Gateway
	IdleTimeout       time.Duration
	Lookups           Ports // map[<port:string>][<host:string>]*ServerMux
	ReadHeaderTimeout time.Duration
	ListenPort        int
	UseXFH            bool
	WorkDir           string
}

// DefaultConfig sets some defaults for the ingress HTTP server.
var DefaultConfig = &Config{
	ConfigFile:        "couper.hcl",
	IdleTimeout:       time.Second * 60,
	ReadHeaderTimeout: time.Second * 10,
	ListenPort:        8080,
	UseXFH:            false,
	WorkDir:           "",
}

// Configure configurates the ingress HTTP server.
func Configure(conf *Config, logger *logrus.Entry) {
	if conf == nil {
		return
	}

	if err := undateByENV(conf); err != nil {
		logger.Fatal(err)
	}

	if err := configureWorkDir(conf); err != nil {
		logger.Fatal(err)
	}

	if err := readHCL(conf); err != nil {
		logger.Fatal(err)
	}
}

func configureWorkDir(conf *Config) error {
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

func readHCL(conf *Config) error {
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
