package accesscontrol_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/internal/test"
)

func Test_NewTokenSource(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	sExpr := hcl.StaticExpr(cty.StringVal("s"), hcl.Range{})

	type testCase struct {
		name    string
		bearer  bool
		cookie  string
		header  string
		value   hcl.Expression
		expType ac.TokenSourceType
		expName string
		expExpr hcl.Expression
	}

	for _, tc := range []testCase{
		{
			"default", false, "", "", nilExpr, ac.BearerType, "", nil,
		},
		{
			"bearer", true, "", "", nilExpr, ac.BearerType, "", nil,
		},
		{
			"cookie", false, "c", "", nilExpr, ac.CookieType, "c", nil,
		},
		{
			"header", false, "", "h", nilExpr, ac.HeaderType, "h", nil,
		},
		{
			"value", false, "", "", sExpr, ac.ValueType, "", sExpr,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			ts, err := ac.NewTokenSource(tc.bearer, tc.cookie, tc.header, tc.value)
			helper.Must(err)

			if ts.Type != tc.expType {
				subT.Errorf("expected token source type: %v, got: %v", tc.expType, ts.Type)
			}
			if ts.Name != tc.expName {
				subT.Errorf("expected token source name: %q, got: %q", tc.expName, ts.Name)
			}
			if ts.Expr != tc.expExpr {
				subT.Errorf("expected token source expr: %#v, got: %#v", tc.expExpr, ts.Expr)
			}
		})
	}
}

func Test_NewTokenSource_error(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	sExpr := hcl.StaticExpr(cty.StringVal("s"), hcl.Range{})

	type testCase struct {
		name   string
		bearer bool
		cookie string
		header string
		value  hcl.Expression
	}

	for _, tc := range []testCase{
		{
			"bearer + cookie", true, "c", "", nilExpr,
		},
		{
			"bearer + header", true, "", "h", nilExpr,
		},
		{
			"bearer + value", true, "", "", sExpr,
		},
		{
			"cookie + header", false, "c", "h", nilExpr,
		},
		{
			"cookie + value", false, "c", "", sExpr,
		},
		{
			"header + value", false, "", "h", sExpr,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			ts, err := ac.NewTokenSource(tc.bearer, tc.cookie, tc.header, tc.value)
			if ts != nil {
				subT.Fatal("expected nil token source")
			}
			if err == nil {
				subT.Fatal("expected error")
			}
			if err.Error() != "only one of bearer, cookie, header or token_value attributes is allowed" {
				subT.Errorf("wrong error message: %q", err.Error())
			}
		})
	}
}
