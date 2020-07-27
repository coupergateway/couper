package config

import (
	"github.com/hashicorp/hcl/v2"
)

type Backend struct {
	Hostname       string   `hcl:"hostname,optional"`
	Name           string   `hcl:"name,label"`
	Origin         string   `hcl:"origin"`
	Path           string   `hcl:"path,optional"`
	Timeout        string   `hcl:"timeout,optional"`
	ConnectTimeout string   `hcl:"connect_timeout,optional"`
	Options        hcl.Body `hcl:",remain"`
}

var (
	backendDefaultTimeout        = "60s"
	backendDefaultConnectTimeout = "10s"
)

// Merge overrides the left backend configuration and returns a new instance.
func (b *Backend) Merge(other *Backend) *Backend {
	if b == nil || other == nil {
		return nil
	}

	result := *b

	if other.Hostname != "" {
		result.Hostname = other.Hostname
	}

	if other.Name != "" {
		result.Name = other.Name
	}

	if other.Origin != "" {
		result.Origin = other.Origin
	}

	if other.Path != "" {
		result.Path = other.Path
	}

	if other.Options != nil {
		result.Options = other.Options
	}

	if other.Timeout != "" {
		result.Timeout = other.Timeout
	}

	if other.ConnectTimeout != "" {
		result.ConnectTimeout = other.ConnectTimeout
	}

	return &result
}
