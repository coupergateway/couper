package config

import (
	"github.com/hashicorp/hcl/v2"
)

const (
	ServeDir  = "ServeDir"
	ServeFile = "ServeFile"
)

type Backend struct {
	Name        string   `hcl:"name,label"`
	Description string   `hcl:"description,optional"`
	Hostname    string   `hcl:"hostname,optional"`
	Origin      string   `hcl:"origin"`
	Path        string   `hcl:"path,optional"`
	Options     hcl.Body `hcl:",remain"`
}
