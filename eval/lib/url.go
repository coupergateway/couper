package lib

import (
	"net/url"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	UrlEncodeFunc = newUrlEncodeFunction()
)

func newUrlEncodeFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "s",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			first := args[0]
			result := url.QueryEscape(first.AsString())
			return cty.StringVal(string(result)), nil
		},
	})
}
