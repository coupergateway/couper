package configload

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
)

func loadSettings(couperConfig *config.Couper, blocks hcl.Blocks) error {
	for _, block := range blocks {
		switch block.Type {
		case settings:
			if diags := gohcl.DecodeBody(block.Body, couperConfig.Context, couperConfig.Settings); diags.HasErrors() {
				return diags
			}

			break
		}
	}

	return nil
}
