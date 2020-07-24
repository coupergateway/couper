package config

import (
	"github.com/hashicorp/hcl/v2"
)

const (
	ServeDir  = "ServeDir"
	ServeFile = "ServeFile"
)

type Backend struct {
	Hostname string   `hcl:"hostname,optional"`
	Name     string   `hcl:"name,label"`
	Origin   string   `hcl:"origin"`
	Path     string   `hcl:"path,optional"`
	Options  hcl.Body `hcl:",remain"`
}

// Merge overrides the left backend configuration and returns a new instance.
func (b *Backend) Merge(other *Backend) *Backend {
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

	return &result
}
