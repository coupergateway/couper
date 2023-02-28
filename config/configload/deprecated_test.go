package configload

import (
	"testing"

	"github.com/avenga/couper/config/parser"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Test_deprecated(t *testing.T) {
	// Insert test data:
	deprecatedAttributes["couper_test_attribute"] = deprecated{"couper_new_attribute", ""}
	deprecatedBlocks["couper_test_block"] = deprecated{"couper_new_block", ""}
	deprecatedLabels["couper_test_label"] = deprecated{"couper_new_label", ""}

	src := []byte(`
couper_test_block "x" "couper_test_label" "abc couper_test_label def" "y" {
	couper_test_attribute = true
}
`)

	body, err := parser.Load(src, "test.go")
	if err != nil {
		t.Fatalf("%s", err)
	}

	deprecate([]*hclsyntax.Body{body})

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

	if body.Blocks[0].Type != "couper_new_block" {
		t.Errorf("Expected 'couper_new_block' block name, got '%s'", body.Blocks[0].Type)
	}

	expLabels := []string{"x", "couper_new_label", "abc", "couper_new_label", "def", "y"}
	if !cmp.Equal(expLabels, body.Blocks[0].Labels) {
		t.Errorf("Expected\n%#v, got:\n%#v", expLabels, body.Blocks[0].Labels)
	}
}
