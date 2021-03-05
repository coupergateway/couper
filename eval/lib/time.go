package lib

import (
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	UnixtimeFunc = newUnixtimeFunction()
)

func newUnixtimeFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.Number),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			return cty.NumberIntVal(time.Now().Unix()), nil
		},
	})
}
