package configload

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
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
	deprecatedBlocks["beta_rate_limit"] = deprecated{"throttle", "1.15"}

	// Deprecated labels:
	deprecatedLabels["beta_backend_rate_limit_exceeded"] = deprecated{"backend_throttle_exceeded", "1.15"}
}

func deprecate(bodies []*hclsyntax.Body, logger *logrus.Entry) {
	for _, body := range bodies {
		deprecateBody(body, logger)
	}
}

func deprecateBody(body *hclsyntax.Body, logger *logrus.Entry) {
	if body == nil {
		return
	}

	body.Attributes = deprecateAttributes(body.Attributes, logger)

	deprecateBlocks(body.Blocks, logger)
}

func deprecateAttributes(attributes hclsyntax.Attributes, logger *logrus.Entry) hclsyntax.Attributes {
	attrs := make(hclsyntax.Attributes)

	for _, attr := range attributes {
		name := attr.Name

		if rename, exists := deprecatedAttributes[name]; exists {
			rename.log("attribute", name, logger)

			name = rename.newName
		}

		attrs[name] = attr
	}

	return attrs
}

func deprecateBlocks(blocks hclsyntax.Blocks, logger *logrus.Entry) {
	for _, block := range blocks {
		block.Labels = deprecateLabels(block, logger)

		if rename, exists := deprecatedBlocks[block.Type]; exists {
			rename.log("block", block.Type, logger)

			block.Type = rename.newName
		}

		deprecateBody(block.Body, logger)
	}
}

func deprecateLabels(block *hclsyntax.Block, logger *logrus.Entry) []string {
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
			rename.log("label", label, logger)

			name = rename.newName
		}

		renamed = append(renamed, name)
	}

	return renamed
}

// In some test cases the logger is <nil>, but not in production code.
func (d deprecated) log(name, old string, logger *logrus.Entry) {
	if logger != nil {
		logger.Warnf(
			"replacing %s %q with %q; as of Couper version %s, the old value is no longer supported",
			name, old, d.newName, d.version,
		)
	}
}
