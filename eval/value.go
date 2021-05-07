package eval

import (
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type ValueFunc interface {
	Value(*hcl.EvalContext, hcl.Expression) (cty.Value, hcl.Diagnostics)
}

// Value is used to populate a child of the given hcl context with the given expression.
// Lookup referenced variables and build up a cty object path to ensure previous attributes
// are there for further lookups. Effectively results in cty.NilVal.
//
// A common case would be accessing a deeper nested structure which MAY be incomplete.
// This context population prevents returning unknown cty.Value's which could not be processed.
func Value(ctx *hcl.EvalContext, exp hcl.Expression) (cty.Value, hcl.Diagnostics) {
	vars := exp.Variables()
	populated := ctx.NewChild()

	if len(vars) > 0 {
		populated.Variables = make(map[string]cty.Value)
	}

	for _, traversal := range vars {
		var previousAttr string
		for i, tr := range traversal {
			switch t := tr.(type) {
			case hcl.TraverseRoot:
				if root, exist := ctx.Variables[t.Name]; exist {
					populated.Variables[t.Name] = root
					continue
				}
				// fallback, provide empty obj to populate with nil attrs if required
				populated.Variables[t.Name] = cty.EmptyObjectVal
			case hcl.TraverseAttr:
				// Root element should always be an Object defined with related Context methods.
				// A possible panic if this is not a ValueMap is the correct answer.
				// Known root variables are verified on load.
				parentObj := populated.Variables[traversal.RootName()].AsValueMap()
				if _, exist := parentObj[t.Name]; exist {
					previousAttr = t.Name
					continue
				}

				current := make(map[string]cty.Value)
				if !lastItem(i, traversal) {
					current[t.Name] = cty.EmptyObjectVal
				} else {
					if parentObj[previousAttr].CanIterateElements() {
						current = parentObj[previousAttr].AsValueMap()
						if _, exist := current[t.Name]; !exist {
							current[t.Name] = cty.NilVal
						}
					} else {
						current[t.Name] = cty.NilVal
					}
				}

				parentObj[previousAttr] = cty.ObjectVal(current)
				populated.Variables[traversal.RootName()] = cty.ObjectVal(parentObj)

				previousAttr = t.Name
			case hcl.TraverseIndex:
				parentObj := populated.Variables[traversal.RootName()].AsValueMap()
				current := parentObj[previousAttr]
				if !current.CanIterateElements() {
					// create empty based on key type
					switch t.Key.Type() {
					case cty.Number:
						current = cty.ListValEmpty(t.Key.Type())
					case cty.String:
						current = cty.EmptyObjectVal
					default:
						current = cty.EmptyTupleVal
					}
				}
				parentObj[previousAttr] = current
				populated.Variables[traversal.RootName()] = cty.ObjectVal(parentObj)
			default:
				panic(reflect.TypeOf(tr))
				// TODO: case splat, split, ... :
			}
		}
	}

	return exp.Value(populated)
}

// lastItem just checks a common scenario: last one within a loop.
func lastItem(idx int, trs hcl.Traversal) bool {
	return idx+1 == len(trs)
}
