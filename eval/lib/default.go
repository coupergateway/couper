package lib

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

var DefaultFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{
		Name:             "vals",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		var argTypes []cty.Type
		for _, val := range args {
			// ignore NilType values when determining the unsafe-unified return type
			if val.Type() == cty.NilType {
				continue
			}
			argTypes = append(argTypes, val.Type())
		}
		if len(argTypes) == 0 {
			// no non-NilVal arguments
			return cty.NilType, nil
		}
		retType, _ := convert.UnifyUnsafe(argTypes)
		if retType == cty.NilType {
			return cty.NilType, fmt.Errorf("all defined arguments must have the same type")
		}
		return retType, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		for _, argVal := range args {
			if !argVal.IsKnown() {
				return cty.UnknownVal(retType), nil
			}
			if argVal.IsNull() || argVal.Type() == cty.NilType {
				continue
			}

			// If the fallback type is a string and this argument too but an empty one, consider them as unset.
			if argVal.Type() == cty.String && argVal.AsString() == "" && retType == cty.String {
				continue
			}

			return convert.Convert(argVal, retType)
		}
		return args[len(args)-1], nil
	},
})
