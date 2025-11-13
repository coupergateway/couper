package main

import (
	"strings"
	"testing"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestRenameBetaBlocks_ValidInputs(t *testing.T) {
	body := &hclsyntax.Body{
		Blocks: []*hclsyntax.Block{
			{Type: "beta_foo"},
			{Type: "beta_foo"},
			{Type: "other"},
		},
	}

	err := renameBetaBlocks(body, "beta_foo", "foo")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if body.Blocks[0].Type != "foo" {
		t.Errorf("expected first block to be renamed to 'foo', got: %q", body.Blocks[0].Type)
	}
	if body.Blocks[1].Type != "foo" {
		t.Errorf("expected second block to be renamed to 'foo', got: %q", body.Blocks[1].Type)
	}
	if body.Blocks[2].Type != "other" {
		t.Errorf("expected other block to remain 'other', got: %q", body.Blocks[2].Type)
	}
}

func TestRenameBetaBlocks_NilBody(t *testing.T) {
	err := renameBetaBlocks(nil, "beta_foo", "foo")
	if err != nil {
		t.Fatalf("expected no error for nil body, got: %v", err)
	}
}

func TestRenameBetaBlocks_EmptyFrom(t *testing.T) {
	err := renameBetaBlocks(&hclsyntax.Body{}, "", "foo")
	if err == nil {
		t.Fatal("expected error for empty from, got nil")
	}
	if !strings.Contains(err.Error(), "from cannot be empty") {
		t.Errorf("expected error about empty from, got: %v", err)
	}
}

func TestRenameBetaBlocks_MissingBetaPrefix(t *testing.T) {
	err := renameBetaBlocks(&hclsyntax.Body{}, "foo", "foo")
	if err == nil {
		t.Fatal("expected error when from doesn't start with 'beta_', got nil")
	}
	if !strings.Contains(err.Error(), "must start with 'beta_'") {
		t.Errorf("expected error about missing prefix, got: %v", err)
	}
}

func TestRenameBetaBlocks_IncorrectTo(t *testing.T) {
	err := renameBetaBlocks(&hclsyntax.Body{}, "beta_foo", "bar")
	if err == nil {
		t.Fatal("expected error when to is incorrect, got nil")
	}
	if !strings.Contains(err.Error(), "to must be") {
		t.Errorf("expected error about incorrect to value, got: %v", err)
	}
}

func TestRenameBetaBlocks_ComplexBetaPrefix(t *testing.T) {
	body := &hclsyntax.Body{
		Blocks: []*hclsyntax.Block{
			{Type: "beta_resource_type"},
		},
	}

	err := renameBetaBlocks(body, "beta_resource_type", "resource_type")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if body.Blocks[0].Type != "resource_type" {
		t.Errorf("expected block to be renamed to 'resource_type', got: %q", body.Blocks[0].Type)
	}
}
