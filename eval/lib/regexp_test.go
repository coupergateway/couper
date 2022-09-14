package lib_test

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/test"
)

func TestRegexpSplit(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server {}`), "couper.hcl")
	helper.Must(err)

	hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

	regexpSplitFn := hclContext.Functions["regexp_split"]

	tests := []struct {
		name string
		args []cty.Value
		want cty.Value
	}{
		{
			"comma surrounded by whitespace",
			[]cty.Value{
				cty.StringVal("\\s*,\\s*"),
				cty.StringVal("\ta\t , \t\tb \tb,   c12 ,d,e  "),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("\ta"),
				cty.StringVal("b \tb"),
				cty.StringVal("c12"),
				cty.StringVal("d"),
				cty.StringVal("e  "),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := test.New(subT)

			splitV, serr := regexpSplitFn.Call(tt.args)
			h.Must(serr)
			if !splitV.RawEquals(tt.want) {
				subT.Errorf("Wrong return value:\nwant:\t%#v\ngot:\t%#v\n", tt.want, splitV)
			}
		})
	}
}
