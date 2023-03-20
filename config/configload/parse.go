package configload

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	configfile "github.com/avenga/couper/config/configload/file"
)

func parseFile(filePath string, srcBytes *[][]byte) (*hclsyntax.Body, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	*srcBytes = append(*srcBytes, src)

	parsed, diags := hclparse.NewParser().ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, diags
	}

	return parsed.Body.(*hclsyntax.Body), nil
}

func parseFiles(files configfile.Files) ([]*hclsyntax.Body, [][]byte, error) {
	var (
		srcBytes     [][]byte
		parsedBodies []*hclsyntax.Body
	)

	for _, file := range files {
		if file.IsDir {
			childBodies, bytes, err := parseFiles(file.Children)
			if err != nil {
				return nil, bytes, err
			}

			parsedBodies = append(parsedBodies, childBodies...)
			srcBytes = append(srcBytes, bytes...)
		} else {
			body, err := parseFile(file.Path, &srcBytes)
			if err != nil {
				return nil, srcBytes, err
			}
			parsedBodies = append(parsedBodies, body)
		}
	}

	return parsedBodies, srcBytes, nil
}
