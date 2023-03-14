package accesscontrol_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/internal/test"
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

func Test_TokenValue(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	sExpr := hcl.StaticExpr(cty.StringVal("asdf"), hcl.Range{})

	type testCase struct {
		name      string
		bearer    bool
		cookie    string
		header    string
		value     hcl.Expression
		reqHeader http.Header
		expToken  string
		expErrMsg string
	}

	for _, tc := range []testCase{
		{
			"default", false, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer asdf"}}, "asdf", "",
		},
		{
			"default, missing authorization header", false, "", "", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"default, missing token in bearer authorization header", false, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer "}}, "", "token required",
		},
		{
			"default, different auth scheme", false, "", "", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", "bearer with token required in authorization header",
		},
		{
			"bearer", true, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer asdf"}}, "asdf", "",
		},
		{
			"bearer, missing authorization header", true, "", "", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"bearer, missing token in bearer authorization header", true, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer "}}, "", "token required",
		},
		{
			"bearer, different auth scheme", true, "", "", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", "bearer with token required in authorization header",
		},
		{
			"cookie", false, "c", "", nilExpr, http.Header{"Cookie": []string{"c=asdf"}}, "asdf", "",
		},
		{
			"cookie, missing c cookie", false, "c", "", nilExpr, http.Header{"Cookie": []string{"foo=bar"}}, "", "token required",
		},
		{
			"header", false, "", "h", nilExpr, http.Header{"H": []string{"asdf"}}, "asdf", "",
		},
		{
			"header, missing h header", false, "", "h", nilExpr, http.Header{"Foo": []string{"bar"}}, "", "token required",
		},
		{
			"authorization header", false, "", "authorization", nilExpr, http.Header{"Authorization": []string{"Bearer asdf"}}, "asdf", "",
		},
		{
			"authorization header, missing authorization header", false, "", "authorization", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"authorization header, missing token in bearer authorization header", false, "", "authorization", nilExpr, http.Header{"Authorization": []string{"Bearer "}}, "", "token required",
		},
		{
			"authorization header, different auth scheme", false, "", "authorization", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", "bearer with token required in authorization header",
		},
		{
			"value", false, "", "", sExpr, http.Header{}, "asdf", "",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(tc.bearer, tc.cookie, tc.header, tc.value)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)
			req.Header = tc.reqHeader

			token, err := ts.TokenValue(req)
			if err != nil {
				msg := err.Error()
				if tc.expErrMsg == "" {
					subT.Errorf("expected no error, but got %q", msg)
				} else if tc.expToken != "" {
					subT.Errorf("expected no token, but got %q", token)
				} else if msg != tc.expErrMsg {
					subT.Errorf("expected error message: %q, got %q", tc.expErrMsg, msg)
				}
			} else if token != tc.expToken {
				subT.Errorf("expected token: %q, got %q", tc.expToken, token)
			}
		})
	}
}
