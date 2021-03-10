package lib

import (
	"errors"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	MergeFunc = newMergeFunction()
)

func newMergeFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		VarParam: &function.Parameter{
			Name:             "maps",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
			AllowNull:        true,
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			// empty args is accepted, so assume an empty object since we have no
			// key-value types.
			if len(args) == 0 {
				return cty.EmptyObject, nil
			}
			return cty.DynamicPseudoType, nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
			return merge(args)
		},
	})
}

func merge(args []cty.Value) (cty.Value, error) {
	var t string
	for _, arg := range args {
		if arg.IsNull() {
			continue
		}
		at := arg.Type()
		if at.IsPrimitiveType() {
			return cty.StringVal(""), errors.New("cannot merge primitive value")
		}
		if at.IsObjectType() || at.IsMapType() {
			if t == "" {
				t = "o"
			} else if t != "o" {
				return cty.StringVal(""), errors.New("type mismatch")
			}
		} else if at.IsTupleType() || at.IsListType() {
			if t == "" {
				t = "l"
			} else if t != "l" {
				return cty.StringVal(""), errors.New("type mismatch")
			}
		}
	}
	if t == "o" {
		return mergeObjects(args), nil
	}
	if t == "l" {
		return mergeTuples(args), nil
	}
	return cty.StringVal(""), errors.New("type mismatch")
}

func mergeObjects(args []cty.Value) cty.Value {
	outputMap := make(map[string]cty.Value)
	for _, arg := range args {
		if arg.IsNull() {
			continue
		}
		for it := arg.ElementIterator(); it.Next(); {
			k, v := it.Element()
			if v.IsNull() {
				delete(outputMap, k.AsString())
			} else if existingVal, ok := outputMap[k.AsString()]; !ok {
				// key not set
				outputMap[k.AsString()] = v
			} else if vType := v.Type(); vType.IsPrimitiveType() {
				// primitive type
				outputMap[k.AsString()] = v
			} else if existingValType := existingVal.Type(); existingValType.IsObjectType() && (vType.IsObjectType() || vType.IsMapType()) {
				outputMap[k.AsString()] = mergeObjects([]cty.Value{existingVal, v})
			} else if existingValType.IsTupleType() && (vType.IsTupleType() || vType.IsListType()) {
				outputMap[k.AsString()] = mergeTuples([]cty.Value{existingVal, v})
			} else {
				outputMap[k.AsString()] = v
			}
		}
	}
	return cty.ObjectVal(outputMap)
}

func mergeTuples(args []cty.Value) cty.Value {
	outputList := []cty.Value{}
	for _, arg := range args {
		if arg.IsNull() {
			continue
		}
		for it := arg.ElementIterator(); it.Next(); {
			_, v := it.Element()
			outputList = append(outputList, v)
		}
	}
	return cty.TupleVal(outputList)
}
