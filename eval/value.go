package eval

import (
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/errors"
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
func Value(ctx *hcl.EvalContext, exp hcl.Expression) (cty.Value, error) {
	vars := exp.Variables()
	populated := ctx.NewChild()

	if len(vars) > 0 {
		populated.Variables = make(map[string]cty.Value)
	}

	for _, traversal := range vars {
		traverseSplit := traversal.SimpleSplit()
		rootTraverse := traverseSplit.Abs[0].(hcl.TraverseRoot) // TODO: check for abs ?
		name := rootTraverse.Name

		if root, exist := ctx.Variables[name]; exist {
			// prefer already configured populated root variables
			if _, exist = populated.Variables[name]; !exist {
				populated.Variables[name] = root
			}
		} else {
			// fallback, provide empty obj to populate with nil attrs if required
			populated.Variables[name] = cty.EmptyObjectVal
		}

		// Root element should always be an Object defined with related Context methods.
		// A possible panic if this is not a ValueMap is the correct answer.
		// Known root variables are verified on load.
		populated.Variables[name] = walk(populated.Variables[name], ctx.Variables[name], traverseSplit.Rel)
	}

	v, diags := exp.Value(populated)
	if diags.HasErrors() {
		return v, errors.Evaluation.With(diags)
	}
	return v, nil
}

func walk(variables, parentVariables cty.Value, traversal hcl.Traversal) cty.Value {
	if len(traversal) == 0 {
		return variables
	}

	hasNext := len(traversal) > 1
	currentFn := func(name string) (current cty.Value, exist bool) {
		if parentVariables.CanIterateElements() {
			if current, exist = parentVariables.AsValueMap()[name]; exist {
				// prefer already configured populated root variables
				if variables.CanIterateElements() {
					if c, e := variables.AsValueMap()[name]; e {
						current = c
					}
				}
			}
		}
		return current, exist
	}

	switch t := traversal[0].(type) {
	case hcl.TraverseAttr:
		current, exist := currentFn(t.Name)
		if !exist {
			if hasNext {
				current = cty.EmptyObjectVal
			} else { // last one
				current = cty.NilVal
			}
		} else if hasNext && !current.CanIterateElements() {
			current = cty.EmptyObjectVal
		}

		vars := variables.AsValueMap()
		if vars == nil {
			vars = make(map[string]cty.Value)
		}
		vars[t.Name] = current

		if hasNext {
			// value map content could differ in nested structures.
			// At this point prefer our own since we have copied existing values already via currentFn.
			vars[t.Name] = walk(current, current, traversal[1:])
		}
		return cty.ObjectVal(vars)
	case hcl.TraverseIndex:
		current := variables

		if current.HasIndex(t.Key).True() {
			if hasNext {
				return walk(current, current, traversal[1:])
			}
			return current
		}

		switch t.Key.Type() {
		case cty.Number:
			fidx := t.Key.AsBigFloat()
			idx, _ := fidx.Int64()
			slice := make([]cty.Value, idx+1)
			if hasNext {
				slice[idx] = cty.EmptyTupleVal
				val := cty.TupleVal(slice)
				return walk(val, val, traversal[1:])
			}
			return cty.TupleVal(slice)
		case cty.String:
			m := map[string]cty.Value{}
			if hasNext {
				m[t.Key.AsString()] = cty.EmptyObjectVal
				val := cty.ObjectVal(m)
				return walk(val, val, traversal[1:])
			} else {
				m[t.Key.AsString()] = cty.NilVal
				return cty.ObjectVal(m)
			}
		default:
			panic(reflect.TypeOf(t))
		}
	default:
		panic(reflect.TypeOf(t))
	}
}
