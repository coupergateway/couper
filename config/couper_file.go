package config

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
)

const DefaultFileName = "couper.hcl"

type CouperFile struct {
	Bytes       []byte
	Context     *hcl.EvalContext
	Definitions *Definitions `hcl:"definitions,block"`
	Server      []*Server    `hcl:"server,block"`
	Settings    *Settings    `hcl:"settings,block"`
}

func SetWorkingDirectory(configFile string) (string, error) {
	if err := os.Chdir(filepath.Dir(configFile)); err != nil {
		return "", err
	}
	return os.Getwd()
}
