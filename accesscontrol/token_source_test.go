package accesscontrol_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/avenga/couper/accesscontrol"
)

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
