package configload

import (
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/parser"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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

		parsedBodies = append(parsedBodies, hclBody)
		srcs = append(srcs, tc.src)
	}

	return bodiesToConfig(parsedBodies, srcs, "")
}
