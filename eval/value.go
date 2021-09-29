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
	expr := exp
	if val := newLiteralValueExpr(ctx, exp); val != nil {
		expr = val
	}

	v, diags := expr.Value(ctx)
	if diags.HasErrors() {
		return v, errors.Evaluation.With(diags)
	}
	return v, nil
}

func newLiteralValueExpr(ctx *hcl.EvalContext, exp hcl.Expression) hclsyntax.Expression {
	switch expr := exp.(type) {
	case *hclsyntax.ConditionalExpr:
		if val := newLiteralValueExpr(ctx, expr.Condition); val != nil {
			expr.Condition = val
		}
		// conditional results must not be cty.NilVal
		if val := newLiteralValueExpr(ctx, expr.TrueResult); val != nil {
			expr.TrueResult = &hclsyntax.LiteralValueExpr{Val: cty.DynamicVal}
		}
		if val := newLiteralValueExpr(ctx, expr.FalseResult); val != nil {
			expr.FalseResult = &hclsyntax.LiteralValueExpr{Val: cty.DynamicVal}
		}
		return expr
	case *hclsyntax.BinaryOpExpr:
		if val := newLiteralValueExpr(ctx, expr.LHS); val != nil {
			expr.LHS = val
		}
		if val := newLiteralValueExpr(ctx, expr.RHS); val != nil {
			expr.RHS = val
		}
		return expr
	case *hclsyntax.ObjectConsExpr:
		for i, item := range expr.Items {
			if val := newLiteralValueExpr(ctx, item.ValueExpr); val != nil {
				expr.Items[i].ValueExpr = val
			}
		}
		return expr
	case *hclsyntax.TemplateExpr:
		for p, part := range expr.Parts {
			for _, v := range part.Variables() {
				if traversalValue(ctx.Variables, v) == cty.NilVal {
					expr.Parts[p] = &hclsyntax.LiteralValueExpr{Val: emptyStringVal}
					break
				}
			}
		}
		return expr
	case *hclsyntax.TemplateWrapExpr:
		expr.Wrapped = newLiteralValueExpr(ctx, expr.Wrapped)
		return expr
	case *hclsyntax.ScopeTraversalExpr:
		if traversalValue(ctx.Variables, expr.Traversal) == cty.NilVal {
			return &hclsyntax.LiteralValueExpr{Val: cty.NilVal}
		}
		return expr
	case *hclsyntax.FunctionCallExpr:
		for a, arg := range expr.Args {
			for _, v := range arg.Variables() {
				if val := traversalValue(ctx.Variables, v); val == cty.NilVal {
					expr.Args[a] = &hclsyntax.LiteralValueExpr{Val: val}
					break
				}
			}
		}
		return expr
	default:
		for _, v := range exp.Variables() {
			if val := traversalValue(ctx.Variables, v); val == cty.NilVal {
				return &hclsyntax.LiteralValueExpr{Val: val}
			}
		}
		return nil
	}
}

func walk(variables, fallback cty.Value, traversal hcl.Traversal) cty.Value {
	if len(traversal) == 0 {
		return variables
	}

	hasNext := len(traversal) > 1
	nextValue := fallback

	currentFn := func(name string) (current cty.Value, exist bool) {
		if variables.CanIterateElements() {
			current, exist = variables.AsValueMap()[name]
		}
		return current, exist
	}

	switch t := traversal[0].(type) {
	case hcl.TraverseAttr:
		current, exist := currentFn(t.Name)
		if !exist {
			return nextValue
		}
		return walk(current, fallback, traversal[1:])

	case hcl.TraverseIndex:
		if variables.HasIndex(t.Key).True() {
			if hasNext {
				return walk(variables, fallback, traversal[1:])
			}
			return variables
		}

		switch t.Key.Type() {
		case cty.Number:
			fidx := t.Key.AsBigFloat()
			idx, _ := fidx.Int64()
			slice := make([]cty.Value, idx+1)
			slice[idx] = nextValue
			if hasNext {
				slice[idx] = walk(nextValue, fallback, traversal[1:])
			}
			return cty.TupleVal(slice)
		case cty.String:
			m := map[string]cty.Value{}
			m[t.Key.AsString()] = nextValue
			if hasNext {
				m[t.Key.AsString()] = walk(nextValue, fallback, traversal[1:])
			}
			return cty.ObjectVal(m)
		default:
			panic(reflect.TypeOf(t))
		}
	default:
		panic(reflect.TypeOf(t))
	}
}

func traversalValue(vars map[string]cty.Value, traversal hcl.Traversal) cty.Value {
	traverseSplit := traversal.SimpleSplit()
	rootTraverse := traverseSplit.Abs[0].(hcl.TraverseRoot) // TODO: check for abs ?
	name := rootTraverse.Name

	if _, ok := vars[name]; !ok {
		return cty.NilVal
	}

	return walk(vars[name], cty.NilVal, traverseSplit.Rel)
}
