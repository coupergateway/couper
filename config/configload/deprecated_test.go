package configload

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/internal/test"
)

func Test_deprecated(t *testing.T) {
	// Insert test data:
	deprecatedAttributes["couper_test_attribute"] = deprecated{"couper_new_attribute", "1.23"}
	deprecatedBlocks["couper_test_block"] = deprecated{"couper_new_block", "1.23"}
	deprecatedLabels["couper_test_label"] = deprecated{"couper_new_label", "1.23"}

	src := []byte(`
error_handler "x" "couper_test_label" "abc couper_test_label def" "y" {
	couper_test_attribute = true
	couper_test_block {
	}
}
`)

	body, err := parser.Load(src, "test.go")
	if err != nil {
		t.Fatalf("%s", err)
	}

	logger, hook := test.NewLogger()
	hook.Reset()

	deprecate([]*hclsyntax.Body{body}, logger.WithContext(context.TODO()))

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

	entries := hook.AllEntries()
	if len(entries) != 4 {
		t.Fatal("Unexpected number of log entries given")
	}

	exp := `replacing label "couper_test_label" with "couper_new_label"; as of Couper version 1.23, the old value is no longer supported`
	if entries[0].Message != exp {
		t.Errorf("Expected\n%#v, got:\n%#v", exp, entries[0].Message)
	}
	if entries[1].Message != exp {
		t.Errorf("Expected\n%#v, got:\n%#v", exp, entries[0].Message)
	}

	exp = `replacing attribute "couper_test_attribute" with "couper_new_attribute"; as of Couper version 1.23, the old value is no longer supported`
	if entries[2].Message != exp {
		t.Errorf("Expected\n%#v, got:\n%#v", exp, entries[0].Message)
	}

	exp = `replacing block "couper_test_block" with "couper_new_block"; as of Couper version 1.23, the old value is no longer supported`
	if entries[3].Message != exp {
		t.Errorf("Expected\n%#v, got:\n%#v", exp, entries[0].Message)
	}
}
