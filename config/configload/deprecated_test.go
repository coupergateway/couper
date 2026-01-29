package configload

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/parser"
	"github.com/coupergateway/couper/internal/test"
)

func Test_deprecated(t *testing.T) {
	// Clear any warnings from previous tests
	ClearWarnings()

	// Insert test data:
	deprecatedAttributes["couper_test_attribute"] = deprecated{newName: "couper_new_attribute", version: "1.23"}
	deprecatedBlocks["couper_test_block"] = deprecated{newName: "couper_new_block", version: "1.23"}
	deprecatedLabels["couper_test_label"] = deprecated{newName: "couper_new_label", version: "1.23"}

	src := []byte(`
error_handler "x" "couper_test_label" "abc couper_test_label def" "y" {
	couper_test_attribute = true
	couper_test_block {
	}
}
`)

	body, err := parser.Load(src, "test.hcl")
	if err != nil {
		t.Fatalf("%s", err)
	}

	deprecate([]*hclsyntax.Body{body})

	// Verify renaming was applied
	if len(body.Blocks) != 1 {
		t.Fatal("Unexpected number of blocks given")
	}

	if len(body.Blocks[0].Body.Attributes) != 1 {
		t.Fatal("Unexpected number of attributes given")
	}

	if attr, exists := body.Blocks[0].Body.Attributes["couper_new_attribute"]; !exists {
		t.Error("Missing 'couper_new_attribute' attribute")
	} else if attr.Name != "couper_test_attribute" {
		t.Errorf("Unexpected attribute name given: '%s'", attr.Name)
	}

	if body.Blocks[0].Body.Blocks[0].Type != "couper_new_block" {
		t.Errorf("Expected 'couper_new_block' block name, got '%s'", body.Blocks[0].Type)
	}

	expLabels := []string{"x", "couper_new_label", "abc", "couper_new_label", "def", "y"}
	if !cmp.Equal(expLabels, body.Blocks[0].Labels) {
		t.Errorf("Expected\n%#v, got:\n%#v", expLabels, body.Blocks[0].Labels)
	}

	// Verify warnings were collected with locations
	warnings := GetWarnings()
	if len(warnings) != 4 {
		t.Fatalf("Expected 4 warnings, got %d", len(warnings))
	}

	// Check warning details
	labelWarnings := 0
	attrWarnings := 0
	blockWarnings := 0
	for _, w := range warnings {
		switch w.Kind {
		case "label":
			labelWarnings++
			if w.OldName != "couper_test_label" || w.NewName != "couper_new_label" {
				t.Errorf("Unexpected label warning: %+v", w)
			}
		case "attribute":
			attrWarnings++
			if w.OldName != "couper_test_attribute" || w.NewName != "couper_new_attribute" {
				t.Errorf("Unexpected attribute warning: %+v", w)
			}
		case "block":
			blockWarnings++
			if w.OldName != "couper_test_block" || w.NewName != "couper_new_block" {
				t.Errorf("Unexpected block warning: %+v", w)
			}
		}
		// Verify location is populated
		if w.Location == "" {
			t.Errorf("Expected location to be populated for %s warning", w.Kind)
		}
		if w.Version != "1.23" {
			t.Errorf("Expected version 1.23, got %s", w.Version)
		}
	}

	if labelWarnings != 2 {
		t.Errorf("Expected 2 label warnings, got %d", labelWarnings)
	}
	if attrWarnings != 1 {
		t.Errorf("Expected 1 attribute warning, got %d", attrWarnings)
	}
	if blockWarnings != 1 {
		t.Errorf("Expected 1 block warning, got %d", blockWarnings)
	}

	// Test EmitWarnings with logger
	logger, hook := test.NewLogger()
	hook.Reset()

	EmitWarnings(logger.WithContext(context.TODO()))

	entries := hook.AllEntries()
	if len(entries) != 4 {
		t.Fatalf("Expected 4 log entries, got %d", len(entries))
	}

	// Verify log messages include location and deprecation info
	for _, entry := range entries {
		if !strings.Contains(entry.Message, "test.hcl:") {
			t.Errorf("Expected log message to contain file location, got: %s", entry.Message)
		}
		if !strings.Contains(entry.Message, "deprecated") {
			t.Errorf("Expected log message to contain 'deprecated', got: %s", entry.Message)
		}
		if !strings.Contains(entry.Message, "please use") {
			t.Errorf("Expected log message to contain 'please use', got: %s", entry.Message)
		}
		if !strings.Contains(entry.Message, "1.23") {
			t.Errorf("Expected log message to contain version, got: %s", entry.Message)
		}
	}

	// Verify warnings were cleared after emit
	warnings = GetWarnings()
	if len(warnings) != 0 {
		t.Errorf("Expected warnings to be cleared after emit, got %d", len(warnings))
	}

	// Clean up test data
	delete(deprecatedAttributes, "couper_test_attribute")
	delete(deprecatedBlocks, "couper_test_block")
	delete(deprecatedLabels, "couper_test_label")
}

func Test_deprecated_contextAware(t *testing.T) {
	// Clear any warnings from previous tests
	ClearWarnings()

	// Add context-aware deprecation: only deprecate inside "parent_block"
	deprecatedBlocks["context_test_block"] = deprecated{
		newName:   "new_context_block",
		version:   "1.99",
		parentCtx: "parent_block",
	}

	src := []byte(`
# This should NOT be renamed (wrong parent context)
context_test_block "outside" {
}

# This should be renamed (correct parent context)
parent_block {
	context_test_block "inside" {
	}
}
`)

	body, err := parser.Load(src, "context_test.hcl")
	if err != nil {
		t.Fatalf("%s", err)
	}

	deprecate([]*hclsyntax.Body{body})

	// Verify only the block inside parent_block was renamed
	if len(body.Blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(body.Blocks))
	}

	// First block should NOT be renamed (outside parent context)
	if body.Blocks[0].Type != "context_test_block" {
		t.Errorf("Expected first block to remain 'context_test_block', got '%s'", body.Blocks[0].Type)
	}

	// Second block is parent_block, check its child
	if body.Blocks[1].Type != "parent_block" {
		t.Fatalf("Expected second block to be 'parent_block', got '%s'", body.Blocks[1].Type)
	}

	if len(body.Blocks[1].Body.Blocks) != 1 {
		t.Fatalf("Expected 1 child block in parent_block, got %d", len(body.Blocks[1].Body.Blocks))
	}

	// Child block should be renamed
	if body.Blocks[1].Body.Blocks[0].Type != "new_context_block" {
		t.Errorf("Expected child block to be renamed to 'new_context_block', got '%s'",
			body.Blocks[1].Body.Blocks[0].Type)
	}

	// Verify only one warning was generated
	warnings := GetWarnings()
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning (for context-aware rename), got %d", len(warnings))
	}

	// Clean up
	ClearWarnings()
	delete(deprecatedBlocks, "context_test_block")
}

func Test_deprecated_betaJob(t *testing.T) {
	// Clear any warnings from previous tests
	ClearWarnings()

	// Test the actual beta_job -> job deprecation
	src := []byte(`
server {}

definitions {
	beta_job "test" {
		interval = "1m"
		request {}
	}
}
`)

	body, err := parser.Load(src, "beta_job_test.hcl")
	if err != nil {
		t.Fatalf("%s", err)
	}

	deprecate([]*hclsyntax.Body{body})

	// Find the definitions block and verify beta_job was renamed
	var defsBlock *hclsyntax.Block
	for _, block := range body.Blocks {
		if block.Type == "definitions" {
			defsBlock = block
			break
		}
	}

	if defsBlock == nil {
		t.Fatal("Expected to find definitions block")
	}

	if len(defsBlock.Body.Blocks) != 1 {
		t.Fatalf("Expected 1 block in definitions, got %d", len(defsBlock.Body.Blocks))
	}

	if defsBlock.Body.Blocks[0].Type != "job" {
		t.Errorf("Expected beta_job to be renamed to 'job', got '%s'", defsBlock.Body.Blocks[0].Type)
	}

	// Verify warning was generated
	warnings := GetWarnings()
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}

	if warnings[0].OldName != "beta_job" || warnings[0].NewName != "job" {
		t.Errorf("Unexpected warning: %+v", warnings[0])
	}

	if warnings[0].Version != "1.15" {
		t.Errorf("Expected version 1.15, got %s", warnings[0].Version)
	}

	// Clean up
	ClearWarnings()
}

func Test_deprecated_beta_rate_limit(t *testing.T) {
	ClearWarnings()

	src := []byte(`
backend {
	beta_rate_limit {
		period = "1m"
		per_period = 60
	}
}
`)

	body, err := parser.Load(src, "test.hcl")
	if err != nil {
		t.Fatalf("%s", err)
	}

	deprecate([]*hclsyntax.Body{body})

	if len(body.Blocks) != 1 {
		t.Fatal("Unexpected number of blocks given")
	}

	if len(body.Blocks[0].Body.Blocks) != 1 {
		t.Fatal("Unexpected number of inner blocks given")
	}

	if body.Blocks[0].Body.Blocks[0].Type != "throttle" {
		t.Errorf("Expected 'throttle' block name, got '%s'", body.Blocks[0].Body.Blocks[0].Type)
	}

	warnings := GetWarnings()
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}

	if warnings[0].OldName != "beta_rate_limit" || warnings[0].NewName != "throttle" {
		t.Errorf("Unexpected warning: %+v", warnings[0])
	}

	ClearWarnings()
}

func Test_deprecated_beta_backend_rate_limit_exceeded(t *testing.T) {
	ClearWarnings()

	src := []byte(`
error_handler "beta_backend_rate_limit_exceeded" {
}
`)

	body, err := parser.Load(src, "test.hcl")
	if err != nil {
		t.Fatalf("%s", err)
	}

	deprecate([]*hclsyntax.Body{body})

	if len(body.Blocks) != 1 {
		t.Fatal("Unexpected number of blocks given")
	}

	expLabels := []string{"backend_throttle_exceeded"}
	if !cmp.Equal(expLabels, body.Blocks[0].Labels) {
		t.Errorf("Expected\n%#v, got:\n%#v", expLabels, body.Blocks[0].Labels)
	}

	warnings := GetWarnings()
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}

	if warnings[0].OldName != "beta_backend_rate_limit_exceeded" || warnings[0].NewName != "backend_throttle_exceeded" {
		t.Errorf("Unexpected warning: %+v", warnings[0])
	}

	ClearWarnings()
}
