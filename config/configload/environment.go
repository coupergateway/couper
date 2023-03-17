package configload

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func preprocessEnvironmentBlocks(bodies []*hclsyntax.Body, env string) error {
	found := false
	for _, body := range bodies {
		f, err := preprocessBody(body, env)
		if err != nil {
			return err
		}
		found = found || f
	}

	if found && env == "" {
		return fmt.Errorf(`"environment" blocks found, but "COUPER_ENVIRONMENT" setting is missing`)
	}

	return nil
}

func preprocessBody(body *hclsyntax.Body, env string) (bool, error) {
	var blocks []*hclsyntax.Block
	found := false

	for _, block := range body.Blocks {
		if block.Type != environment {
			blocks = append(blocks, block)
			continue
		}

		found = true

		if len(block.Labels) == 0 {
			defRange := block.DefRange()
			return true, newDiagErr(&defRange, "Missing label(s) for 'environment' block")
		}

		for i, label := range block.Labels {
			if err := validLabel(label, &block.LabelRanges[i]); err != nil {
				return true, err
			}

			if label == env {
				blocks = append(blocks, block.Body.Blocks...)

				for name, attr := range block.Body.Attributes {
					body.Attributes[name] = attr
				}
			}
		}
	}

	for _, block := range blocks {
		foundInChildren, err := preprocessBody(block.Body, env)
		if err != nil {
			return found || foundInChildren, err
		}
		found = found || foundInChildren
	}

	body.Blocks = blocks

	return found, nil
}
