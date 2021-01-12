package parser

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

func Load(src []byte, name string) (hcl.Body, hcl.Diagnostics) {
	parser := hclparse.NewParser()

	var file *hcl.File
	var diags hcl.Diagnostics

	if strings.HasSuffix(name, ".json") {
		file, diags = parser.ParseJSON(src, name)
	} else {
		file, diags = parser.ParseHCL(src, name)
	}

	if file == nil || file.Body == nil {
		return hcl.EmptyBody(), diags
	}
	return file.Body, diags
}
