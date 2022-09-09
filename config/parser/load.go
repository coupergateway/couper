package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Load(src []byte, name string) (*hclsyntax.Body, error) {
	parser := hclparse.NewParser()

	file, diags := parser.ParseHCL(src, name)
	if file == nil || file.Body == nil {
		return nil, diags
	}

	hsbody, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("couper configuration must be in native HCL syntax")
	}
	return hsbody, nil
}
