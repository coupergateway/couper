package configload

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func preprocessEnvironmentBlocks(bodies []*hclsyntax.Body, env string) error {
	for _, body := range bodies {
		if err := preprocessBody(body, env); err != nil {
			return err
		}
	}

	return nil
}

func preprocessBody(parent *hclsyntax.Body, env string) error {
	var blocks []*hclsyntax.Block

	for _, block := range parent.Blocks {
		if block.Type != environment {
			blocks = append(blocks, block)

			continue
		}

		if len(block.Labels) == 0 {
			defRange := block.DefRange()

			return hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing label(s) for 'environment' block",
					Subject:  &defRange,
				},
			}
		}

		for _, label := range block.Labels {
			if err := validLabel(label, getRange(block.Body)); err != nil {
				return err
			}

			if label == env {
				blocks = append(blocks, block.Body.Blocks...)

				for name, attr := range block.Body.Attributes {
					parent.Attributes[name] = attr
				}
			}
		}
	}

	for _, block := range blocks {
		if err := preprocessBody(block.Body, env); err != nil {
			return err
		}
	}

	parent.Blocks = blocks

	return nil
}
