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

	// Deprecated blocks:
	// deprecatedBlocks["..."] = deprecated{"...", "..."}

	// Deprecated labels:
	// deprecatedLabels["..."] = deprecated{"...", "..."}
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
	var renamed []string

	labels, _ := newKindsFromLabels(block)

	for _, label := range labels {
		name := label

		if rename, exists := deprecatedLabels[label]; exists {
			name = rename.newName
		}

		renamed = append(renamed, name)
	}

	return renamed
}
