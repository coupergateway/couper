package body_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/eval"
)

func TestMergeBodies(t *testing.T) {
	type expectedBody struct {
		OAuth2       *config.OAuth2ReqAuth `hcl:"oauth2,block"`
		Request      *config.Request       `hcl:"request,block"`
		TokenRequest *config.Request       `hcl:"token_request,block"`
	}

	type container struct {
		Block []*expectedBody `hcl:"block,block"`
	}

	bodies := &container{[]*expectedBody{
		{OAuth2: &config.OAuth2ReqAuth{
			BackendName: "test",
			GrantType:   "override_me",
		}},
		{OAuth2: &config.OAuth2ReqAuth{
			GrantType: "no_creds",
		}},
	}}

	expectedAttributes := map[string]*hcl.Attribute{
		"backend":        {Name: "backend", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("test")}},
		"grant_type":     {Name: "grant_type", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("no_creds")}},
		"token_endpoint": {Name: "token_endpoint", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("http://that")}},
		"url":            {Name: "url", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("https://that")}},
		"attr1":          {Name: "attr1", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("https://bar")}},
		"attr2":          {Name: "attr2", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("https://the-force")}},
	}

	var hclBodies []hcl.Body

	// could be done with hclmock package, but for example purposes encode and read
	target := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(bodies, target.Body())
	// write config reference
	//buf := &bytes.Buffer{}
	//_, _ = target.WriteTo(buf)

	configBytes := []byte(`
block {

  oauth2 {
    backend        = "test"
    grant_type     = "no_creds"
    token_endpoint = "http://this"
  }

  request "label" {
    url = "https://this"
  }

  request "default" {
    attr1 = "https://foo"
  }

  token_request "default" {
    attr2 = "https://may-"
  }
}

block {

  oauth2 {
    token_endpoint = "http://that"
  }

  request "label" {
    url = "https://that"
  }

  request "default" {
    attr1 = "https://bar"
  }

  token_request "default" {
    attr2 = "https://the-force"
  }
}`)

	conf, diags := hclsyntax.ParseConfig(configBytes, "testcase.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Error(diags)
	}
	defer func() {
		if t.Failed() {
			println(string(configBytes))
		}
	}()

	schema, _ := gohcl.ImpliedBodySchema(&container{})
	content, _, diags := conf.Body.PartialContent(schema)
	if diags.HasErrors() {
		t.Error(diags)
	}
	for _, inner := range content.Blocks.OfType("block") {
		hclBodies = append(hclBodies, inner.Body)
	}

	schema, _ = gohcl.ImpliedBodySchema(bodies.Block[0])
	content, _, diags = body.MergeBodies(hclBodies[:]...).PartialContent(schema)
	if diags.HasErrors() {
		t.Error(diags)
	}

	// after merge, we expect a single block with a body of type mergedBodies
	if len(content.Blocks.OfType("oauth2")) != 1 {
		t.Error("expected just one merged oauth2 block")
	}
	oauthBlockContent := content.Blocks.OfType("oauth2")[0]
	resultAttributes, diags := oauthBlockContent.Body.JustAttributes()
	if diags.HasErrors() {
		t.Error(diags)
	}

	// same applies to labeled ones, after merge, we expect a single block with a body of type mergedBodies
	if len(content.Blocks.OfType("token_request")) != 1 {
		t.Error("expected one merged token_request block")
	}
	if len(content.Blocks.OfType("request")) != 2 {
		t.Error("expected two merged request blocks")
	}

	for _, requestBlockContent := range append(content.Blocks.OfType("request"),
		content.Blocks.OfType("token_request")...) {
		requestBlockAttrs, diags := requestBlockContent.Body.JustAttributes()
		if diags.HasErrors() {
			t.Error(diags)
		}
		// caution; block attrs must differ (unique attribute names)
		for k, v := range requestBlockAttrs {
			resultAttributes[k] = v
		}
	}

	hclcontext := eval.NewDefaultContext().HCLContext()

	for k, attr := range expectedAttributes {
		a, exist := resultAttributes[k]
		if !exist {
			t.Errorf("missing attribute %q", k)
			continue
		}

		expVal, diags := attr.Expr.Value(hclcontext)
		if diags.HasErrors() {
			t.Error(diags)
		}

		val, diags := a.Expr.Value(hclcontext)
		if diags.HasErrors() {
			t.Error(diags)
		}

		if expVal.AsString() != val.AsString() {
			t.Errorf("Want: %q, got %q", expVal.AsString(), val.AsString())
		}
	}
}

func Test_stringSliceEquals(t *testing.T) {
	type args struct {
		left  []string
		right []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"join glue", args{[]string{"a", "ab"}, []string{"aa", "b"}}, false},
		{"join", args{[]string{"aa", "bb"}, []string{"bb", "aa"}}, true},
		{"join", args{[]string{"aa", "bb"}, nil}, false},
		{"join", args{nil, []string{"aa", "bb"}}, false},
		{"join", args{nil, nil}, true},
		{"join", args{[]string{""}, []string{"", ""}}, false},
		{"join", args{[]string{"", ""}, []string{"", ""}}, true},
		{"join", args{[]string{""}, []string{""}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := body.StringSliceEquals(tt.args.left, tt.args.right); got != tt.want {
				t.Errorf("StringSliceEquals() = %v, want %v", got, tt.want)
			}
		})
	}
}
