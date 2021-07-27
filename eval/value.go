package eval

import (
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/errors"
)

type ValueFunc interface {
	Value(*hcl.EvalContext, hcl.Expression) (cty.Value, hcl.Diagnostics)
}

var emptyStringVal = cty.StringVal("")

// Value is used to populate a child of the given hcl context with the given expression.
// Lookup referenced variables and build up a cty object path to ensure previous attributes
// are there for further lookups. Effectively results in cty.NilVal.
//
// A common case would be accessing a deeper nested structure which MAY be incomplete.
// This context population prevents returning unknown cty.Value's which could not be processed.
func Value(ctx *hcl.EvalContext, exp hcl.Expression) (cty.Value, error) {
	vars := exp.Variables()
	populated := ctx.NewChild()
	tplVars := templateVariables(exp) // Remember template expression related variables.

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

		// Value which should be set at traversal path end.
		fallback := cty.NilVal
		if isTemplateTraversal(traverseSplit, tplVars) {
			// Additionally, check for template expression usage which would fail
			// with cty.NilVal while receiving the expression value. Use an empty string instead.
			fallback = emptyStringVal
		}

		// Root element should always be an Object defined with related Context methods.
		// A possible panic if this is not a ValueMap is the correct answer.
		// Known root variables are verified on load.
		populated.Variables[name] = walk(populated.Variables[name], ctx.Variables[name], fallback, traverseSplit.Rel)
	}

	v, diags := exp.Value(populated)
	if diags.HasErrors() {
		return v, errors.Evaluation.With(diags)
	}

	return finalize(v), nil
}

func walk(variables, parentVariables, fallback cty.Value, traversal hcl.Traversal) cty.Value {
	if len(traversal) == 0 {
		return variables
	}

	hasNext := len(traversal) > 1
	nextValue := fallback
	if hasNext {
		switch traversal[1].(type) {
		case hcl.TraverseIndex, hcl.TraverseSplat:
			nextValue = cty.EmptyTupleVal
		default:
			nextValue = cty.EmptyObjectVal
		}
	}

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
			current = nextValue
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
			vars[t.Name] = walk(current, current, fallback, traversal[1:])
		}
		return cty.ObjectVal(vars)
	case hcl.TraverseIndex:
		current := variables

		if current.HasIndex(t.Key).True() {
			if hasNext {
				return walk(current, current, fallback, traversal[1:])
			}
			return current
		}

		switch t.Key.Type() {
		case cty.Number:
			fidx := t.Key.AsBigFloat()
			idx, _ := fidx.Int64()
			slice := make([]cty.Value, idx+1)
			slice[idx] = nextValue
			if hasNext {
				slice[idx] = walk(nextValue, nextValue, fallback, traversal[1:])
			}
			return cty.TupleVal(slice)
		case cty.String:
			m := map[string]cty.Value{}
			m[t.Key.AsString()] = nextValue
			if hasNext {
				m[t.Key.AsString()] = walk(nextValue, nextValue, fallback, traversal[1:])
			}
			return cty.ObjectVal(m)
		default:
			panic(reflect.TypeOf(t))
		}
	default:
		panic(reflect.TypeOf(t))
	}
}

func templateVariables(exp hcl.Expression) (vars []hcl.Traversal) {
	objExp, ok := exp.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return vars
	}

	// ObjValueMap could have expressions within the key and expression. Lookup both.
	for _, item := range objExp.Items {
		switch keyItem := item.KeyExpr.(type) {
		case *hclsyntax.ObjectConsKeyExpr:
			if _, tplOk := keyItem.Wrapped.(*hclsyntax.TemplateWrapExpr); tplOk {
				vars = append(vars, keyItem.Variables()...)
			}
		}

		switch item.ValueExpr.(type) {
		case *hclsyntax.TemplateExpr:
			vars = append(vars, item.ValueExpr.Variables()...)
		}
	}

	return vars
}

func isTemplateTraversal(split hcl.TraversalSplit, tplVars []hcl.Traversal) bool {
	for _, tplVar := range tplVars {
		if split.RootName() != tplVar.RootName() {
			continue
		}

		tplSplit := tplVar.SimpleSplit()
		if len(split.Rel) != len(tplSplit.Rel) {
			continue
		}

		result := true
		for i, r := range tplSplit.Rel {
			rattr, a := r.(hcl.TraverseAttr)
			iattr, b := split.Rel[i].(hcl.TraverseAttr)
			if (!a || !b) || iattr.Name != rattr.Name {
				result = false
				break
			}
		}

		if result {
			return true
		}

	}

	return false
}

// finalize will modify the given cty.Value if the corresponding key value
// is a cty.Map with cty.NilVal's. This map will be replaced with a cty.NilVal.
//
// This is necessary for populated "nil paths" which have shared nested references.
func finalize(v cty.Value) cty.Value {
	if !v.CanIterateElements() {
		return v
	}

	vmap := v.AsValueMap()
	for k, mv := range vmap {
		if !mv.CanIterateElements() {
			continue
		}

		nonNilStop := mv.ForEachElement(isNilElement)
		if !nonNilStop {
			vmap[k] = cty.NilVal
		}
	}
	return cty.ObjectVal(vmap)
}

// isNilElement is used as iterate callback and returns true if a non NilVal gets passed.
func isNilElement(_ cty.Value, val cty.Value) (stop bool) {
	if val.Equals(cty.NilVal) == cty.BoolVal(false) {
		return true
	}
	return false
}
