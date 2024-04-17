package lib_test

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/test"
)

func TestDefaultErrors(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server {}`), "couper.hcl")
	helper.Must(err)

	tests := []struct {
		name    string
		args    []cty.Value
		wantErr string
	}{
		{
			"mixed types",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("1"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.NumberIntVal(1),
				}),
			},
			"all defined arguments must have the same type",
		},
	}

	hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := hclContext.Functions["default"].Call(tt.args)
			if err == nil {
				subT.Error("Error expected")
			}
			if err != nil && err.Error() != tt.wantErr {
				subT.Errorf("Wrong error message; expected %#v, got: %#v", tt.wantErr, err.Error())
			}
		})
	}
}
