package configload

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func Test_absPath(t *testing.T) {
	tests := []struct {
		name string
		attr *hclsyntax.Attribute
		want cty.Value
	}{
		{"absolute path",
			&hclsyntax.Attribute{Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("/www")}, SrcRange: hcl.Range{Filename: "/dir1/case1.hcl"}},
			cty.StringVal("/www"),
		},
		{"relative path",
			&hclsyntax.Attribute{Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("./www")}, SrcRange: hcl.Range{Filename: "/dir2/case2.hcl"}},
			cty.StringVal("/dir2/www"),
		},
		{"relative parent dir path",
			&hclsyntax.Attribute{Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("./../subDir1/www")}, SrcRange: hcl.Range{Filename: "/dir3/case3.hcl"}},
			cty.StringVal("/subDir1/www"),
		},
		{"relative parent dir path 2",
			&hclsyntax.Attribute{Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("./../../../../../subDir2/www")}, SrcRange: hcl.Range{Filename: "/dir4/case4.hcl"}},
			cty.StringVal("/subDir2/www"),
		},
		{"relative path w/o dot",
			&hclsyntax.Attribute{Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("www")}, SrcRange: hcl.Range{Filename: "/dir5/case5.hcl"}},
			cty.StringVal("/dir5/www"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := absPath(tt.attr)
			gotExpr := got.(*hclsyntax.LiteralValueExpr)
			gotValue, _ := gotExpr.Value(envContext)
			if gotValue.Equals(tt.want).False() {
				t.Errorf("absPath() = %q, want %q", gotValue.AsString(), tt.want.AsString())
			}
		})
	}
}
