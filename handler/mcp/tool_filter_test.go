package mcp

import "testing"

func TestToolFilter_IsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		filter   ToolFilter
		tool     string
		expected bool
	}{
		{"empty filter allows all", ToolFilter{}, "anything", true},
		{"allowed exact match", ToolFilter{Allowed: []string{"get_weather"}}, "get_weather", true},
		{"allowed exact no match", ToolFilter{Allowed: []string{"get_weather"}}, "delete_file", false},
		{"allowed wildcard", ToolFilter{Allowed: []string{"read_*"}}, "read_file", true},
		{"allowed wildcard no match", ToolFilter{Allowed: []string{"read_*"}}, "write_file", false},
		{"blocked exact match", ToolFilter{Blocked: []string{"delete_file"}}, "delete_file", false},
		{"blocked exact no match", ToolFilter{Blocked: []string{"delete_file"}}, "read_file", true},
		{"blocked wildcard", ToolFilter{Blocked: []string{"delete_*"}}, "delete_table", false},
		{"blocked wildcard no match", ToolFilter{Blocked: []string{"delete_*"}}, "read_table", true},
		{
			"allowed and blocked - allowed wins for match",
			ToolFilter{Allowed: []string{"read_*", "delete_file"}, Blocked: []string{"delete_*"}},
			"read_file",
			true,
		},
		{
			"allowed and blocked - blocked overrides allowed",
			ToolFilter{Allowed: []string{"read_*", "delete_*"}, Blocked: []string{"delete_*"}},
			"delete_file",
			false,
		},
		{"multiple allowed patterns", ToolFilter{Allowed: []string{"read_*", "search_*", "get_weather"}}, "search_code", true},
		{"multiple blocked patterns", ToolFilter{Blocked: []string{"delete_*", "exec_*"}}, "exec_command", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsAllowed(tt.tool)
			if result != tt.expected {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.tool, result, tt.expected)
			}
		})
	}
}

func TestToolFilter_FilterTools(t *testing.T) {
	tools := []Tool{
		{Name: "read_file"},
		{Name: "write_file"},
		{Name: "delete_file"},
		{Name: "search_code"},
		{Name: "get_weather"},
	}

	filter := ToolFilter{
		Allowed: []string{"read_*", "get_*", "search_*"},
		Blocked: []string{"search_*"},
	}

	result := filter.FilterTools(tools)

	expected := map[string]bool{"read_file": true, "get_weather": true}
	if len(result) != len(expected) {
		t.Fatalf("expected %d tools, got %d", len(expected), len(result))
	}
	for _, tool := range result {
		if !expected[tool.Name] {
			t.Errorf("unexpected tool %q in result", tool.Name)
		}
	}
}

func TestToolFilter_InvalidGlobPattern_FailsClosed(t *testing.T) {
	filter := ToolFilter{Allowed: []string{"[invalid"}}
	if filter.IsAllowed("anything") {
		t.Error("invalid allowed pattern should fail closed (deny)")
	}

	filter2 := ToolFilter{Blocked: []string{"[invalid"}}
	if filter2.IsAllowed("anything") {
		t.Error("invalid blocked pattern should fail closed (deny)")
	}
}

func TestToolFilter_HasRules(t *testing.T) {
	if (&ToolFilter{}).HasRules() {
		t.Error("empty filter should have no rules")
	}
	if !(&ToolFilter{Allowed: []string{"a"}}).HasRules() {
		t.Error("filter with allowed should have rules")
	}
	if !(&ToolFilter{Blocked: []string{"b"}}).HasRules() {
		t.Error("filter with blocked should have rules")
	}
}

func TestToolFilter_FilterTools_Empty(t *testing.T) {
	tools := []Tool{{Name: "a"}, {Name: "b"}}
	filter := ToolFilter{}

	result := filter.FilterTools(tools)
	if len(result) != 2 {
		t.Fatalf("empty filter should return all tools, got %d", len(result))
	}
}
