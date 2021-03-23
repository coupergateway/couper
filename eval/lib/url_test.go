package lib_test

import (
	"testing"

	"github.com/avenga/couper/eval"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/internal/test"
)

func TestUrlEncode(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server "test" {}`), "couper.hcl")
	helper.Must(err)

	hclContext := cf.Context.Value(eval.ContextType).(*eval.Context).HCLContext()

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
