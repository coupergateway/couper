package body_test

import (
	"reflect"
	"testing"

	"github.com/avenga/couper/config/body"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"
)

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
