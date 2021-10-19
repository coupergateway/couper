package lib

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

var CoalesceFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{
		Name:             "vals",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) < 2 {
			return cty.NilType, fmt.Errorf("not enough arguments")
		}
		// last argument defines the impl return type
		return args[len(args)-1].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		for _, argVal := range args {
			if !argVal.IsKnown() {
				return cty.UnknownVal(retType), nil
			}
			if argVal.IsNull() || argVal.Type() == cty.NilType {
				continue
			}

			return convert.Convert(argVal, retType)
		}
		return cty.NilVal, fmt.Errorf("no non-null or nil-type arguments")
	},
})
