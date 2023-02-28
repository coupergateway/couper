package configload

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type deprecated struct {
	newName string
	version string
}

type (
	attributesList map[string]deprecated
	blocksList     map[string]deprecated
	labelsList     map[string]deprecated
)

var (
	deprecatedAttributes attributesList
	deprecatedBlocks     blocksList
	deprecatedLabels     labelsList
)

func init() {
	deprecatedAttributes = make(map[string]deprecated)
	deprecatedBlocks = make(map[string]deprecated)
	deprecatedLabels = make(map[string]deprecated)

	// Deprecated attributes:
	// deprecatedAttributes["..."] = deprecated{"...", "..."}
	deprecatedAttributes["beta_permissions_claim"] = deprecated{"permissions_claim", "1.13"}
	deprecatedAttributes["beta_permissions_map"] = deprecated{"permissions_map", "1.13"}
	deprecatedAttributes["beta_permissions_map_file"] = deprecated{"permissions_map_file", "1.13"}
	deprecatedAttributes["beta_required_permission"] = deprecated{"required_permission", "1.13"}
	deprecatedAttributes["beta_roles_claim"] = deprecated{"roles_claim", "1.13"}
	deprecatedAttributes["beta_roles_map"] = deprecated{"roles_map", "1.13"}
	deprecatedAttributes["beta_roles_file"] = deprecated{"roles_map_file", "1.13"}

	// Deprecated blocks:
	// deprecatedBlocks["..."] = deprecated{"...", "..."}

	// Deprecated labels:
	// deprecatedLabels["..."] = deprecated{"...", "..."}
	deprecatedLabels["beta_insufficient_permissions"] = deprecated{"insufficient_permissions", "1.13"}
}

func deprecate(bodies []*hclsyntax.Body) {
	for _, body := range bodies {
		deprecateBody(body)
	}
}

func deprecateBody(body *hclsyntax.Body) {
	if body == nil {
		return
	}

	body.Attributes = deprecateAttributes(body.Attributes)

	deprecateBlocks(body.Blocks)
}

func deprecateAttributes(attributes hclsyntax.Attributes) hclsyntax.Attributes {
	attrs := make(hclsyntax.Attributes)

	for _, attr := range attributes {
		name := attr.Name

		if rename, exists := deprecatedAttributes[name]; exists {
			name = rename.newName
		}

		attrs[name] = attr
	}

	return attrs
}

func deprecateBlocks(blocks hclsyntax.Blocks) {
	for _, block := range blocks {
		block.Labels = deprecateLabels(block)

		if rename, exists := deprecatedBlocks[block.Type]; exists {
			block.Type = rename.newName
		}

		deprecateBody(block.Body)
	}
}

func deprecateLabels(block *hclsyntax.Block) []string {
	var (
		err     error
		labels  []string = block.Labels
		renamed []string
	)

	if block.Type == errorHandler {
		labels, err = newKindsFromLabels(block, false)

		if err != nil {
			return block.Labels
		}
	}

	for _, label := range labels {
		name := label

		if rename, exists := deprecatedLabels[label]; exists {
			name = rename.newName
		}

		renamed = append(renamed, name)
	}

	return renamed
}
