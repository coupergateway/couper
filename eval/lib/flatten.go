package lib

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var FlattenFunc = newFlattenFunction()

func newFlattenFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name:             "list",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
			AllowNull:        true,
		}},
		Type: function.StaticReturnType(cty.DynamicPseudoType),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			val := args[0]
			if val.IsNull() {
				return cty.EmptyTupleVal, nil
			}

			if !val.CanIterateElements() {
				return cty.NilVal, fmt.Errorf("flatten requires a list or tuple, got %s", val.Type().FriendlyName())
			}

			var result []cty.Value
			flattenValue(val, &result)

			if len(result) == 0 {
				return cty.EmptyTupleVal, nil
			}

			return cty.TupleVal(result), nil
		},
	})
}

func flattenValue(val cty.Value, result *[]cty.Value) {
	if val.IsNull() || !val.IsKnown() {
		return
	}

	ty := val.Type()
	if !ty.IsTupleType() && !ty.IsListType() && !ty.IsSetType() {
		*result = append(*result, val)
		return
	}

	for it := val.ElementIterator(); it.Next(); {
		_, v := it.Element()
		flattenValue(v, result)
	}
}
