package configload

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
)

func loadDefinitions(couperConfig *config.Couper, blocks hcl.Blocks) (Backends, error) {
	var backends Backends

	for _, block := range blocks {
		switch block.Type {
		case definitions:
			backendContent, leftOver, diags := block.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return nil, diags
			}

			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					name := be.Labels[0]

					ref, _ := backends.WithName(name)
					if ref != nil {
						return nil, hcl.Diagnostics{&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  fmt.Sprintf("duplicate backend name: %q", name),
							Subject:  &be.LabelRanges[0],
						}}
					}

					backends = append(backends, NewBackend(name, be.Body))
				}
			}

			if diags = gohcl.DecodeBody(leftOver, couperConfig.Context, couperConfig.Definitions); diags.HasErrors() {
				return nil, diags
			}

			break
		}
	}

	return backends, nil
}
