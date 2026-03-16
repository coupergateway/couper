package mcp

import "path"

// ToolFilter applies allow/block glob patterns to tool names.
type ToolFilter struct {
	Allowed []string
	Blocked []string
}

// IsAllowed checks if a tool name passes the filter.
// If no filter patterns are set, all tools are allowed.
func (f *ToolFilter) IsAllowed(toolName string) bool {
	if len(f.Allowed) == 0 && len(f.Blocked) == 0 {
		return true
	}

	allowed := true
	if len(f.Allowed) > 0 {
		allowed = false
		for _, pattern := range f.Allowed {
			if matched, _ := path.Match(pattern, toolName); matched {
				allowed = true
				break
			}
		}
	}

	if !allowed {
		return false
	}

	for _, pattern := range f.Blocked {
		if matched, _ := path.Match(pattern, toolName); matched {
			return false
		}
	}

	return true
}

// FilterTools filters a slice of tools, returning only allowed ones.
func (f *ToolFilter) FilterTools(tools []Tool) []Tool {
	if len(f.Allowed) == 0 && len(f.Blocked) == 0 {
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
