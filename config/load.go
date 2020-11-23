package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclsimple"

	"github.com/avenga/couper/eval"
)

func LoadFile(filePath string) (*Gateway, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	src, err := ioutil.ReadFile(path.Join(wd, filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return LoadBytes(src, filePath)
}

func LoadBytes(src []byte, filePath string) (*Gateway, error) {
	config := &Gateway{
		Context:  eval.NewENVContext(src),
		Settings: &Settings{DefaultPort: DefaultListenPort},
	}
	filename := filepath.Base(filePath)
	if filepath.Ext(filename) != ".hcl" {
		return nil, fmt.Errorf("configuration must be a hcl file")
	}

	// filename must match .hcl ending for further []byte processing
	if err := hclsimple.Decode(filename, src, config.Context, config); err != nil {
		return nil, fmt.Errorf("Failed to load configuration bytes: %w", err)
	}
	return config, nil
}
