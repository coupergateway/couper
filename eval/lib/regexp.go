package lib

import (
	"regexp"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var RegexpSplitFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "pattern",
			Type: cty.String,
		},
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		pattern := args[0].AsString()
		re, err := regexp.Compile(pattern)
		if err != nil {
			return cty.UnknownVal(retType), err
		}
		str := args[1].AsString()
		elems := re.Split(str, -1)
		elemVals := make([]cty.Value, len(elems))
		for i, s := range elems {
			elemVals[i] = cty.StringVal(s)
		}
		if len(elemVals) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}
		return cty.ListVal(elemVals), nil
	},
})
