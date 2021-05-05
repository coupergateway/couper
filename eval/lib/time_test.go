package lib_test

import (
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/test"
)

func TestUnixtime(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server "test" {}`), "couper.hcl")
	helper.Must(err)

	hclContext := cf.Context.Value(eval.ContextType).(*eval.Context).HCLContext()

	expectedNow := time.Now().Unix()
	now, err := hclContext.Functions["unixtime"].Call([]cty.Value{})
	helper.Must(err)

	if !cty.Number.Equals(now.Type()) {
		t.Errorf("Wrong return type; expected %s, got: %s", cty.Number.FriendlyName(), now.Type().FriendlyName())
	}

	bfnow := now.AsBigFloat()
	inow, _ := bfnow.Int64()
	if !fuzzyEqual(expectedNow, inow, 2) {
		t.Errorf("Wrong return value; expected %d, got: %d", expectedNow, inow)
	}
}

func fuzzyEqual(a, b, fuzz int64) bool {
	return b <= a+fuzz &&
		b >= a-fuzz
}
