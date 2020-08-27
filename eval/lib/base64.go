package lib

import (
	"encoding/base64"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	Base64DecodeFunc = newBase64DecodeFunction()
	Base64EncodeFunc = newBase64EncodeFunction()
)

func newBase64DecodeFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "base64_decode",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			first := args[0]
			result, err := base64.StdEncoding.DecodeString(first.AsString())
			if err != nil {
				return cty.StringVal(""), err
			}
			return cty.StringVal(string(result)), nil
		},
	})
}

func newBase64EncodeFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "base64_encode",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			first := args[0]
			result := base64.StdEncoding.EncodeToString([]byte(first.AsString()))
			return cty.StringVal(result), nil
		},
	})
}
