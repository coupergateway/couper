package config

import (
	"context"
	"github.com/avenga/couper/config/configload/file"
)

// DefaultFilename defines the default filename for a couper config file.
const DefaultFilename = "couper.hcl"

// Couper represents the <Couper> config object.
type Couper struct {
	Context     context.Context
	Filename    string
	Files       file.Files
	Definitions *Definitions `hcl:"definitions,block"`
	Servers     Servers      `hcl:"server,block"`
	Settings    *Settings    `hcl:"settings,block"`
	Defaults    *Defaults    `hcl:"defaults,block"`
}
