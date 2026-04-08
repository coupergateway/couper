package mcp

import "path"

// ToolFilter applies allow/block glob patterns to tool names.
type ToolFilter struct {
	Allowed []string
	Blocked []string
}

// HasRules returns true if any filter patterns are configured.
func (f *ToolFilter) HasRules() bool {
	return len(f.Allowed) > 0 || len(f.Blocked) > 0
}

// IsAllowed checks if a tool name passes the filter.
// If no filter patterns are set, all tools are allowed.
// Invalid glob patterns fail closed (deny).
func (f *ToolFilter) IsAllowed(toolName string) bool {
	if !f.HasRules() {
		return true
	}

	allowed := true
	if len(f.Allowed) > 0 {
		allowed = false
		for _, pattern := range f.Allowed {
			matched, err := path.Match(pattern, toolName)
			if err != nil {
				// Invalid pattern — fail closed
				return false
			}
			if matched {
				allowed = true
				break
			}
		}
	}

	if !allowed {
		return false
	}

	for _, pattern := range f.Blocked {
		matched, err := path.Match(pattern, toolName)
		if err != nil {
			// Invalid pattern — fail closed
			return false
		}
		if matched {
			return false
		}
	}

	return true
}

// FilterTools filters a slice of tools, returning only allowed ones.
func (f *ToolFilter) FilterTools(tools []Tool) []Tool {
	if !f.HasRules() {
		return tools
	}

	filtered := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		if f.IsAllowed(tool.Name) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
