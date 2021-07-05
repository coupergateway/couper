package config

import (
	"context"
	"github.com/hashicorp/hcl/v2"
)

// DefaultFilename defines the default filename for a couper config file.
const DefaultFilename = "couper.hcl"

// Couper represents the <Couper> config object.
type Couper struct {
	Bytes       []byte
	Context     context.Context
	Filename    string
	Definitions *Definitions `hcl:"definitions,block"`
	Servers     Servers      `hcl:"server,block"`
	Settings    *Settings    `hcl:"settings,block"`
	Defaults    *Defaults    `hcl:"defaults,block"`
}

type DefaultsBlock struct {
	Defaults *Defaults `hcl:"defaults,block"`
	Remain   hcl.Body  `hcl:",remain"`
}
