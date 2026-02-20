package configload

import (
	"sync"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
)

// DeprecationWarning holds information about a deprecated usage found during config loading.
type DeprecationWarning struct {
	Kind     string // "block", "attribute", "label"
	OldName  string
	NewName  string
	Version  string // version when support will be removed
	Location string // file:line from HCL range
}

// deprecated defines a deprecation mapping with optional context constraint.
type deprecated struct {
	newName   string
	version   string
	parentCtx string // optional: only apply when inside this parent block type (empty = global)
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

	// collectedWarnings stores warnings during config load, emitted after logger is ready
	collectedWarnings   []DeprecationWarning
	collectedWarningsMu sync.Mutex
)

func init() {
	deprecatedAttributes = make(map[string]deprecated)
	deprecatedBlocks = make(map[string]deprecated)
	deprecatedLabels = make(map[string]deprecated)

	// Deprecated attributes:
	// deprecatedAttributes["old_name"] = deprecated{newName: "new_name", version: "1.x"}

	// Deprecated blocks:
	// Use parentCtx to constrain deprecation to specific parent block types
	deprecatedBlocks[betaJob] = deprecated{newName: job, version: "1.15", parentCtx: definitions}
	deprecatedBlocks[betaRateLimit] = deprecated{newName: throttle, version: "1.15"}

	// Deprecated labels:
	deprecatedLabels[betaBackendRateLimitExceeded] = deprecated{newName: backendThrottleExceeded, version: "1.15"}
}

// deprecate processes HCL bodies and renames deprecated elements while collecting warnings.
func deprecate(bodies []*hclsyntax.Body) {
	for _, body := range bodies {
		deprecateBody(body, "")
	}
}

// deprecateBody recursively processes a body, tracking the parent block type for context-aware deprecations.
func deprecateBody(body *hclsyntax.Body, parentType string) {
	if body == nil {
		return
	}

	body.Attributes = deprecateAttributes(body.Attributes)
	deprecateBlocks(body.Blocks, parentType)
}

func deprecateAttributes(attributes hclsyntax.Attributes) hclsyntax.Attributes {
	attrs := make(hclsyntax.Attributes)

	for _, attr := range attributes {
		name := attr.Name

		if rename, exists := deprecatedAttributes[name]; exists {
			collectWarning("attribute", name, rename, attr.SrcRange.String())
			name = rename.newName
		}

		attrs[name] = attr
	}

	return attrs
}

func deprecateBlocks(blocks hclsyntax.Blocks, parentType string) {
	for _, block := range blocks {
		block.Labels = deprecateLabels(block)

		if rename, exists := deprecatedBlocks[block.Type]; exists {
			// Check context constraint: only apply if parentCtx is empty or matches current parent
			if rename.parentCtx == "" || rename.parentCtx == parentType {
				collectWarning("block", block.Type, rename, block.TypeRange.String())
				block.Type = rename.newName
			}
		}

		deprecateBody(block.Body, block.Type)
	}
}

func deprecateLabels(block *hclsyntax.Block) []string {
	var (
		err     error
		labels  = block.Labels
		renamed []string
	)

	if block.Type == errorHandler {
		labels, err = newKindsFromLabels(block, false)
		if err != nil {
			return block.Labels
		}
	}

	for i, label := range labels {
		name := label

		if rename, exists := deprecatedLabels[label]; exists {
			location := block.LabelRanges[i].String()
			collectWarning("label", label, rename, location)
			name = rename.newName
		}

		renamed = append(renamed, name)
	}

	return renamed
}

// collectWarning adds a deprecation warning to the collection (thread-safe).
func collectWarning(kind, oldName string, rename deprecated, location string) {
	collectedWarningsMu.Lock()
	defer collectedWarningsMu.Unlock()

	collectedWarnings = append(collectedWarnings, DeprecationWarning{
		Kind:     kind,
		OldName:  oldName,
		NewName:  rename.newName,
		Version:  rename.version,
		Location: location,
	})
}

// EmitWarnings logs all collected deprecation warnings and clears the collection.
// Call this after the logger is initialized.
func EmitWarnings(logger *logrus.Entry) {
	collectedWarningsMu.Lock()
	defer collectedWarningsMu.Unlock()

	for _, w := range collectedWarnings {
		if logger != nil {
			logger.Warnf(
				"%s: %s %q is deprecated, please use %q instead; support will be removed in version %s",
				w.Location, w.Kind, w.OldName, w.NewName, w.Version,
			)
		}
	}

	collectedWarnings = nil
}

// GetWarnings returns a copy of the collected warnings (mainly for testing).
func GetWarnings() []DeprecationWarning {
	collectedWarningsMu.Lock()
	defer collectedWarningsMu.Unlock()

	result := make([]DeprecationWarning, len(collectedWarnings))
	copy(result, collectedWarnings)
	return result
}

// ClearWarnings clears the collected warnings without emitting them (mainly for testing).
func ClearWarnings() {
	collectedWarningsMu.Lock()
	defer collectedWarningsMu.Unlock()

	collectedWarnings = nil
}
