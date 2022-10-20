package body_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/eval"
)

func TestBody_MergeBodies(t *testing.T) {
	tests := []struct {
		name            string
		src             *hclsyntax.Body
		replace         bool
		expAttrs        map[string]string
		expBlocksTotal  int
		expBlocksOfType map[string]int
	}{
		{
			"merge, replace",
			&hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"a": &hclsyntax.Attribute{Name: "a", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("a2")}},
					"c": &hclsyntax.Attribute{Name: "c", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("c")}},
				},
				Blocks: hclsyntax.Blocks{
					&hclsyntax.Block{Type: "a", Labels: []string{}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "b", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "c", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
				},
			},
			true,
			map[string]string{"a": "a2", "b": "b", "c": "c"},
			6,
			map[string]int{"a": 2, "b": 2, "c": 2},
		},
		{
			"merge, no replace",
			&hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"a": &hclsyntax.Attribute{Name: "a", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("a2")}},
					"c": &hclsyntax.Attribute{Name: "c", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("c")}},
				},
				Blocks: hclsyntax.Blocks{
					&hclsyntax.Block{Type: "a", Labels: []string{}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "b", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "c", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
				},
			},
			false,
			map[string]string{"a": "a1", "b": "b", "c": "c"},
			6,
			map[string]int{"a": 2, "b": 2, "c": 2},
		},
		{
			"'merge' with self, replace",
			nil,
			true,
			map[string]string{"a": "a1", "b": "b"},
			3,
			map[string]int{"a": 1, "b": 1, "c": 1},
		},
		{
			"'merge' with self, no replace",
			nil,
			false,
			map[string]string{"a": "a1", "b": "b"},
			3,
			map[string]int{"a": 1, "b": 1, "c": 1},
		},
	}

	hclcontext := eval.NewDefaultContext().HCLContext()

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			dest := &hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"a": &hclsyntax.Attribute{Name: "a", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("a1")}},
					"b": &hclsyntax.Attribute{Name: "b", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("b")}},
				},
				Blocks: hclsyntax.Blocks{
					&hclsyntax.Block{Type: "a", Labels: []string{}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "b", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "c", Labels: []string{""}, Body: &hclsyntax.Body{}},
				},
			}
			src := tt.src
			if tt.src == nil {
				src = dest
			}
			merged := body.MergeBodies(dest, src, tt.replace)
			if len(merged.Attributes) != len(tt.expAttrs) {
				subT.Fatalf("expected %d attributes, was %d", len(tt.expAttrs), len(merged.Attributes))
			}
			for k, expV := range tt.expAttrs {
				attr, set := merged.Attributes[k]
				if !set {
					subT.Errorf("expected attribute %q set", k)
				}
				val, diags := attr.Expr.Value(hclcontext)
				if diags.HasErrors() {
					subT.Error(diags)
				}
				sVal := val.AsString()
				if val.AsString() != expV {
					subT.Errorf("attribute value for %q:\nwant: %q\ngot:  %q", k, expV, sVal)
				}
			}
			if len(merged.Blocks) != tt.expBlocksTotal {
				subT.Fatalf("expected %d blocks, was %d", tt.expBlocksTotal, len(merged.Blocks))
			}
			if len(body.BlocksOfType(merged, "a")) != tt.expBlocksOfType["a"] {
				subT.Errorf("expected %d blocks of type a", tt.expBlocksOfType["a"])
			}
			if len(body.BlocksOfType(merged, "b")) != tt.expBlocksOfType["b"] {
				subT.Errorf("expected %d blocks of type b", tt.expBlocksOfType["b"])
			}
			if len(body.BlocksOfType(merged, "c")) != tt.expBlocksOfType["c"] {
				subT.Errorf("expected %d blocks of type c", tt.expBlocksOfType["c"])
			}
		})
	}
}
