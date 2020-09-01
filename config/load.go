package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/hcl/v2/hclsimple"

	"go.avenga.cloud/couper/gateway/eval"
)

func LoadFile(filename string) (*Gateway, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	src, err := ioutil.ReadFile(path.Join(wd, filename))
	if err != nil {
		return nil, fmt.Errorf("Failed to load configuration: %w", err)
	}
	return LoadBytes(src)
}

func LoadBytes(src []byte) (*Gateway, error) {
	config := &Gateway{Context: eval.NewENVContext(src)}
	// filename must match .hcl ending for further []byte processing
	if err := hclsimple.Decode("loadBytes.hcl", src, config.Context, config); err != nil {
		return nil, fmt.Errorf("Failed to load configuration bytes: %w", err)
	}
	return config, nil
}
