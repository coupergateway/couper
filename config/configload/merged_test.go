package configload_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
)

func TestMergeBodies(t *testing.T) {
	type expectedBody struct {
		OAuth2 *config.OAuth2ReqAuth `hcl:"oauth2,block"`
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
		"backend":    {Name: "backend", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("test")}},
		"grant_type": {Name: "grant_type", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("no_creds")}},
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
}
block {

  oauth2 {
    token_endpoint = "http://that"
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
	content, _, diags = configload.MergeBodies(hclBodies[:]).PartialContent(schema)
	if diags.HasErrors() {
		t.Error(diags)
	}

	oauthBlockContent := content.Blocks.OfType("oauth2")[0]
	resultAttributes, diags := oauthBlockContent.Body.JustAttributes()
	if diags.HasErrors() {
		t.Error(diags)
	}

	hclcontext := eval.NewContext(nil).HCLContext()
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
