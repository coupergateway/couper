package lib_test

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/test"
)

func TestUrlEncode(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server "test" {}`), "couper.hcl")
	helper.Must(err)

	hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

	s := "ABC123abc\n :/?#[]@!$&'()*+,;=%"
	encodedV, err := hclContext.Functions["url_encode"].Call([]cty.Value{cty.StringVal(s)})
	helper.Must(err)

	if !cty.String.Equals(encodedV.Type()) {
		t.Errorf("Wrong return type; expected %s, got: %s", cty.String.FriendlyName(), encodedV.Type().FriendlyName())
	}

	encoded := encodedV.AsString()
	expected := "ABC123abc%0A%20%3A%2F%3F%23%5B%5D%40%21%24%26%27%28%29%2A%2B%2C%3B%3D%25"
	if encoded != expected {
		t.Errorf("Wrong return value; expected %s, got: %s", expected, encoded)
	}
}

func TestRelativeUrl(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server "test" {}`), "couper.hcl")
	helper.Must(err)

	hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()
	relativeUrlFunc := hclContext.Functions["relative_url"]

	type testCase struct {
		url    string
		expURL string
		expErr string
	}

	for _, tc := range []testCase{
		// Invalid
		{"", "", `invalid url given: ''`},
		{"rel", "", `invalid url given: 'rel'`},
		{"?q", "", `invalid url given: '?q'`},
		{"?", "", `invalid url given: '?'`},
		{"#f", "", `invalid url given: '#f'`},
		{"#", "", `invalid url given: '#'`},
		{"~", "", `invalid url given: '~'`},
		{"abc@def.org", "", `invalid url given: 'abc@def.org'`},
		{"ftp://127.0.0.1", "", `invalid url given: 'ftp://127.0.0.1'`},

		// Valid
		{"/abs", "/abs", ``},
		{"/abs:8080", "/abs:8080", ``},
		{"https://abc.def:8443:9443", "/", ``},
		{"http://", "/", ``},
		{"http://abc", "/", ``},
		{"http://abc.def", "/", ``},
		{"http://abc.def?", "/?", ``},
		{"http://abc.def#", "/#", ``},
		{"http://abc.def/#", "/#", ``},
		{"http://abc.def?#", "/?#", ``},
		{"http://abc.def/?#", "/?#", ``},
		{"https://abc.def/path", "/path", ``},
		{"https://abc.def/path?a+b", "/path?a+b", ``},
		{"https://abc.def/path?a%20b", "/path?a%20b", ``},
		{"https://abc.def:8443/path?q", "/path?q", ``},
		{"https://abc.def:8443/path?q#f", "/path?q#f", ``},
		{"https://user:pass@abc.def:8443/path?q#f", "/path?q#f", ``},
	} {
		t.Run(tc.url, func(subT *testing.T) {
			got, err := relativeUrlFunc.Call([]cty.Value{cty.StringVal(tc.url)})

			if tc.expURL != "" && got.AsString() != tc.expURL {
				t.Errorf("'%#v': expected '%s', got '%s'", tc.url, tc.expURL, got.AsString())
			}
			if got != cty.NilVal && tc.expURL == "" {
				t.Errorf("'%#v': expected 'cty.NilVal', got '%s'", tc.url, got.AsString())
			}
			if tc.expErr != "" || err != nil {
				if eerr := fmt.Sprintf("%s", err); tc.expErr != eerr {
					t.Errorf("'%#v': expected '%s', got '%s'", tc.url, tc.expErr, eerr)
				}
			}
		})
	}
}
