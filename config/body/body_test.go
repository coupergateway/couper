package body_test

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/eval"
)

func TestBody_MergeBds(t *testing.T) {
	tests := []struct {
		name     string
		dest     *hclsyntax.Body
		src      *hclsyntax.Body
		replace  bool
		expAttrs map[string]string
	}{
		{
			"merge with replace",
			&hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"a": &hclsyntax.Attribute{Name: "a", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("a1")}},
					"b": &hclsyntax.Attribute{Name: "b", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("b")}},
				},
				Blocks: hclsyntax.Blocks{
					&hclsyntax.Block{Type: "a", Labels: []string{}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "b", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "c", Labels: []string{""}, Body: &hclsyntax.Body{}},
				},
			},
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
		},
		{
			"merge with no replace",
			&hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"a": &hclsyntax.Attribute{Name: "a", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("a1")}},
					"b": &hclsyntax.Attribute{Name: "b", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("b")}},
				},
				Blocks: hclsyntax.Blocks{
					&hclsyntax.Block{Type: "a", Labels: []string{}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "b", Labels: []string{"label"}, Body: &hclsyntax.Body{}},
					&hclsyntax.Block{Type: "c", Labels: []string{""}, Body: &hclsyntax.Body{}},
				},
			},
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
		},
	}

	hclcontext := eval.NewDefaultContext().HCLContext()

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			merged := body.MergeBds(tt.dest, tt.src, tt.replace)
			if len(merged.Attributes) != 3 {
				subT.Fatal("expected 3 attributes")
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
			if len(merged.Blocks) != 6 {
				subT.Fatal("expected 6 blocks")
			}
			if len(body.BlocksOfType(merged, "a")) != 2 {
				subT.Fatal("expected 2 blocks of type a")
			}
			if len(body.BlocksOfType(merged, "b")) != 2 {
				subT.Fatal("expected 2 blocks of type b")
			}
			if len(body.BlocksOfType(merged, "c")) != 2 {
				subT.Fatal("expected 2 blocks of type c")
			}
		})
	}
}

func TestBody_Body(t *testing.T) {
	itemRange := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 1, Column: 2, Byte: 3},
		End:      hcl.Pos{Line: 4, Column: 5, Byte: 6},
	}
	attrs := hcl.Attributes{
		"name": &hcl.Attribute{
			Name: "test",
			Expr: hcltest.MockExprLiteral(cty.StringVal("value")),
		},
	}

	content := &hcl.BodyContent{
		Attributes:       attrs,
		MissingItemRange: itemRange,
	}
	body := body.New(content)

	mir := body.MissingItemRange()
	if !reflect.DeepEqual(mir, itemRange) {
		t.Errorf("want\n%#v\ngot\n%#v", itemRange, mir)
	}

	c, diags := body.Content(nil)
	if diags != nil {
		t.Errorf("Unexpected diags: %#v", diags)
	}
	if !reflect.DeepEqual(c, content) {
		t.Errorf("want\n%#v\ngot\n%#v", content, c)
	}

	p, b, diags := body.PartialContent(nil)
	if diags != nil {
		t.Errorf("Unexpected diags: %#v", diags)
	}
	if !reflect.DeepEqual(p, content) {
		t.Errorf("want\n%#v\ngot\n%#v", content, p)
	}
	if !reflect.DeepEqual(b, body) {
		t.Errorf("want\n%#v\ngot\n%#v", body, b)
	}

	a, diags := body.JustAttributes()
	if diags != nil {
		t.Errorf("Unexpected diags: %#v", diags)
	}
	if !reflect.DeepEqual(a, attrs) {
		t.Errorf("want\n%#v\ngot\n%#v", attrs, a)
	}
}
