package runtime

import "github.com/avenga/couper/config"

type Config struct {
	File      string `env:"config_file"`
	LogFormat string `env:"log_format"`
}

// NewConfig creates the runtime config which could be overridden in order:
// internal.defaults -> config.settings -> flag.args -> env.vars
func NewConfig(c *config.Gateway) *Config {
	defaultConf := *DefaultConfig
	conf := &defaultConf
	if c != nil && c.Settings != nil {
		conf.Merge(newConfigFrom(c.Settings))
	}

	return conf
}

func newConfigFrom(s *config.Settings) *Config {
	return &Config{
		LogFormat: s.LogFormat,
	}
}

func (c *Config) Merge(o *Config) *Config {
	if o == nil {
		return c
	}

	if o.File != "" {
		c.File = o.File
	}

	if o.LogFormat != "" {
		c.LogFormat = o.LogFormat
	}

	return c
}
