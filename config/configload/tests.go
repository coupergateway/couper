package configload

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/parser"
)

type testContent struct {
	filename string
	src      []byte
}

func loadTestContents(tcs []testContent) (*config.Couper, error) {
	var (
		parsedBodies []*hclsyntax.Body
		srcs         [][]byte
	)

	for _, tc := range tcs {
		hclBody, err := parser.Load(tc.src, tc.filename)
		if err != nil {
			return nil, err
		}
		deprecate([]*hclsyntax.Body{hclBody})

		parsedBodies = append(parsedBodies, hclBody)
		srcs = append(srcs, tc.src)
	}

	return bodiesToConfig(parsedBodies, srcs, "")
}

func LoadFile(file, env string) (*config.Couper, error) {
	return LoadFiles([]string{file}, env)
}

func LoadBytes(src []byte, filename string) (*config.Couper, error) {
	return LoadBytesEnv(src, filename, "")
}

func LoadBytesEnv(src []byte, filename, env string) (*config.Couper, error) {
	hclBody, err := parser.Load(src, filename)
	if err != nil {
		return nil, err
	}
	deprecate([]*hclsyntax.Body{hclBody})

	if err = validateBody(hclBody, false); err != nil {
		return nil, err
	}

	return bodiesToConfig([]*hclsyntax.Body{hclBody}, [][]byte{src}, env)
}
