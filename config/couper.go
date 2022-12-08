package config

import (
	"context"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/configload/file"
	"github.com/avenga/couper/config/schema"
)

// DefaultFilename defines the default filename for a couper config file.
const DefaultFilename = "couper.hcl"

// Couper represents the <Couper> config object.
type Couper struct {
	Context     context.Context
	Environment string
	Files       file.Files
	Defaults    *Defaults    `hcl:"defaults,block"`
	Definitions *Definitions `hcl:"definitions,block"`
	Servers     Servers      `hcl:"server,block"`
	Settings    *Settings    `hcl:"settings,block"`
}

func init() {
	couper := Couper{}
	couperSchema, _ := gohcl.ImpliedBodySchema(couper)

	// register block headers and body schema
	for _, block := range couperSchema.Blocks {
		var err error
		b := block

		switch block.Type {
		case "defaults":
			err = schema.Registry.Add(couper, &b, Defaults{})
		case "definitions":
			err = schema.Registry.Add(couper, &b, Definitions{})
		case "server":
			err = schema.Registry.Add(couper, &b, Server{})
		case "settings":
			err = schema.Registry.Add(couper, &b, &Settings{})
		}
		if err != nil {
			panic(err)
		}
	}
}
