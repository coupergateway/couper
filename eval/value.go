package eval

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/errors"
)

type ValueFunc interface {
	Value(*hcl.EvalContext, hcl.Expression) (cty.Value, hcl.Diagnostics)
}

var emptyStringVal = cty.StringVal("")

// ValueFromBodyAttribute lookups the given attribute from given hcl.Body and
// returns cty.NilVal if the attribute is not present.
func ValueFromBodyAttribute(ctx *hcl.EvalContext, body *hclsyntax.Body, name string) (cty.Value, error) {
	if body == nil {
		return cty.NilVal, nil
	}

	attr, ok := body.Attributes[name]
	if !ok {
		return cty.NilVal, nil
	}

	return Value(ctx, attr.Expr)
}

// Value is used to clone and modify the given expression if an expression would make use of
// undefined context variables.
// Effectively results in cty.NilVal or empty string value for template expression.
//
// A common case would be accessing a deeper nested structure which MAY be incomplete.
// This replacement prevents returning unknown cty.Value's which could not be processed.
func Value(ctx *hcl.EvalContext, exp hcl.Expression) (cty.Value, error) {
	expr := exp
	// due to some internal types we could not clone all expressions.
	if _, ok := exp.(hclsyntax.Expression); ok {
		expr = clone(exp)

		// replace hcl expressions with literal ones if there is no context variable reference.
		if val := newLiteralValueExpr(ctx, expr); val != nil {
			expr = val
		}
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
		var cond cty.Value
		if val := newLiteralValueExpr(ctx, expr.Condition); val != nil {
			c, diags := val.Value(ctx)
			// allow only bool predicates
			if c.Type() != cty.Bool || diags.HasErrors() {
				return expr
			}
			cond = c
		}

		// Conditional results must not be cty.NilVal and from same type
		// so evaluate already replaced condition first and return
		// LiteralValueExpr instead of a ConditionalExpr.
		if cond.False() {
			return newLiteralValueExpr(ctx, expr.FalseResult)
		}
		return newLiteralValueExpr(ctx, expr.TrueResult)
	case *hclsyntax.BinaryOpExpr:
		if val := newLiteralValueExpr(ctx, expr.LHS); val != nil {
			expr.LHS = val
		}
		if val := newLiteralValueExpr(ctx, expr.RHS); val != nil {
			expr.RHS = val
		}
		return expr
	case *hclsyntax.ObjectConsKeyExpr:
		if val := newLiteralValueExpr(ctx, expr.Wrapped); val != nil {
			expr.Wrapped = val
		}
		return expr
	case *hclsyntax.ObjectConsExpr:
		for i, item := range expr.Items {
			// KeyExpr can't be NilVal
			for _, v := range item.KeyExpr.Variables() {
				if traversalValue(ctx.Variables, v) == cty.NilVal {
					expr.Items[i].KeyExpr = &hclsyntax.LiteralValueExpr{Val: emptyStringVal, SrcRange: v.SourceRange()}
					break
				}
			}

			if val := newLiteralValueExpr(ctx, item.ValueExpr); val != nil {
				expr.Items[i].ValueExpr = val
			}
		}
		return expr
	case *hclsyntax.TemplateExpr:
		for p, part := range expr.Parts {
			expr.Parts[p] = newLiteralValueExpr(ctx, part)
		}

		// "pre"-evaluate to be able to combine string expressions with empty strings on NilVal result.
		c, _ := expr.Value(ctx)
		if c.IsNull() {
			return &hclsyntax.LiteralValueExpr{Val: emptyStringVal, SrcRange: expr.Range()}
		}
		return &hclsyntax.LiteralValueExpr{Val: c, SrcRange: expr.Range()}
	case *hclsyntax.TemplateWrapExpr:
		if val := newLiteralValueExpr(ctx, expr.Wrapped); val != nil {
			expr.Wrapped = val
		}
		return expr
	case *hclsyntax.ScopeTraversalExpr:
		if traversalValue(ctx.Variables, expr.Traversal) == cty.NilVal {
			return &hclsyntax.LiteralValueExpr{Val: cty.NilVal, SrcRange: expr.SrcRange}
		}
		return expr
	case *hclsyntax.FunctionCallExpr:
		for a, arg := range expr.Args {
			expr.Args[a] = newLiteralValueExpr(ctx, arg)
		}
		return expr
	case *hclsyntax.TupleConsExpr:
		for e, ex := range expr.Exprs {
			expr.Exprs[e] = newLiteralValueExpr(ctx, ex)
		}
		return expr
	case *hclsyntax.ForExpr:
		expr.CollExpr = newLiteralValueExpr(ctx, expr.CollExpr)
		return expr
	case *hclsyntax.ParenthesesExpr:
		expr.Expression = newLiteralValueExpr(ctx, expr.Expression)
		return expr
	case *hclsyntax.SplatExpr:
		expr.Each = newLiteralValueExpr(ctx, expr.Each)
		expr.Source = newLiteralValueExpr(ctx, expr.Source)
		return expr
	case *hclsyntax.IndexExpr:
		if val := newLiteralValueExpr(ctx, expr.Collection); val != nil {
			expr.Collection = val
		}
		if val := newLiteralValueExpr(ctx, expr.Key); val != nil {
			expr.Key = val
		}
		return expr
	case *hclsyntax.RelativeTraversalExpr:
		if val := newLiteralValueExpr(ctx, expr.Source); val != nil {
			expr.Source = val
		}
		return expr
	case *hclsyntax.UnaryOpExpr:
		if val := newLiteralValueExpr(ctx, expr.Val); val != nil {
			expr.Val = val
		}
		return expr
	case *hclsyntax.AnonSymbolExpr:
		return expr
	case *hclsyntax.LiteralValueExpr:
		return expr
	default:
		panic("eval.Value: expression type not supported: " + fmt.Sprint(reflect.TypeOf(expr).String()))
	}
}

func clone(exp hcl.Expression) hclsyntax.Expression {
	switch expr := exp.(type) {
	case *hclsyntax.BinaryOpExpr:
		op := *expr.Op
		ex := &hclsyntax.BinaryOpExpr{
			LHS:      clone(expr.LHS),
			Op:       &op,
			RHS:      clone(expr.RHS),
			SrcRange: expr.SrcRange,
		}
		return ex
	case *hclsyntax.ConditionalExpr:
		ex := &hclsyntax.ConditionalExpr{
			Condition:   clone(expr.Condition),
			FalseResult: clone(expr.FalseResult),
			SrcRange:    expr.SrcRange,
			TrueResult:  clone(expr.TrueResult),
		}
		return ex
	case *hclsyntax.ForExpr:
		ex := *expr
		ex.CollExpr = clone(expr.CollExpr)
		return &ex
	case *hclsyntax.FunctionCallExpr:
		ex := *expr
		ex.Args = make([]hclsyntax.Expression, 0)
		for _, arg := range expr.Args { // just clone args; will be modified, see above
			ex.Args = append(ex.Args, clone(arg))
		}
		return &ex
	case *hclsyntax.ObjectConsExpr:
		ex := *expr
		ex.Items = make([]hclsyntax.ObjectConsItem, len(ex.Items))
		for i, item := range expr.Items {
			ex.Items[i].KeyExpr = clone(item.KeyExpr)
			ex.Items[i].ValueExpr = clone(item.ValueExpr)
		}
		return &ex
	case *hclsyntax.ObjectConsKeyExpr:
		ex := *expr
		ex.Wrapped = clone(ex.Wrapped)
		return &ex
	case *hclsyntax.ParenthesesExpr:
		ex := *expr
		ex.Expression = clone(expr.Expression)
		return &ex
	case *hclsyntax.ScopeTraversalExpr:
		ex := *expr
		return &ex
	case *hclsyntax.TemplateExpr:
		ex := *expr
		ex.Parts = make([]hclsyntax.Expression, len(expr.Parts))
		for i, item := range expr.Parts {
			ex.Parts[i] = clone(item)
		}
		return &ex
	case *hclsyntax.TemplateWrapExpr:
		ex := *expr
		ex.Wrapped = clone(ex.Wrapped)
		return &ex
	case *hclsyntax.LiteralValueExpr:
		ex := *expr
		return &ex
	case *hclsyntax.TupleConsExpr:
		ex := *expr
		ex.Exprs = make([]hclsyntax.Expression, len(expr.Exprs))
		for i, item := range expr.Exprs {
			ex.Exprs[i] = clone(item)
		}
		return &ex
	case *hclsyntax.SplatExpr:
		ex := *expr
		ex.Source = clone(expr.Source)
		return &ex
	case *hclsyntax.IndexExpr:
		ex := &hclsyntax.IndexExpr{
			Collection:   clone(expr.Collection),
			Key:          clone(expr.Key),
			SrcRange:     expr.SrcRange,
			OpenRange:    expr.OpenRange,
			BracketRange: expr.BracketRange,
		}
		return ex
	case *hclsyntax.RelativeTraversalExpr:
		ex := &hclsyntax.RelativeTraversalExpr{
			Source:    clone(expr.Source),
			Traversal: expr.Traversal,
			SrcRange:  expr.SrcRange,
		}
		return ex
	case *hclsyntax.UnaryOpExpr:
		ex := &hclsyntax.UnaryOpExpr{
			Op:          expr.Op,
			Val:         clone(expr.Val),
			SrcRange:    expr.SrcRange,
			SymbolRange: expr.SymbolRange,
		}
		return ex
	case *hclsyntax.AnonSymbolExpr:
		return &hclsyntax.AnonSymbolExpr{SrcRange: expr.SrcRange}
	default:
		panic("eval: unsupported expression: " + reflect.TypeOf(exp).String())
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

// walk goes through the given variables path and returns the fallback value if no variable is set.
func walk(variables, fallback cty.Value, traversal hcl.Traversal) cty.Value {
	if len(traversal) == 0 {
		return variables
	}

	hasNext := len(traversal) > 1

	currentFn := func(name string) (current cty.Value, exist bool) {
		if variables.Type().IsObjectType() || variables.Type().IsMapType() {
			current, exist = variables.AsValueMap()[name]
		}
		return current, exist
	}

	switch t := traversal[0].(type) {
	case hcl.TraverseAttr:
		current, exist := currentFn(t.Name)
		if !exist {
			return fallback
		}
		return walk(current, fallback, traversal[1:])
	case hcl.TraverseIndex:
		if !variables.CanIterateElements() {
			return fallback
		}

		switch t.Key.Type() {
		case cty.Number:
			if variables.HasIndex(t.Key).True() {
				if hasNext {
					fidx := t.Key.AsBigFloat()
					idx, _ := fidx.Int64()
					return walk(variables.AsValueSlice()[idx], fallback, traversal[1:])
				}
				return variables
			}
		case cty.String:
			current, exist := currentFn(t.Key.AsString())
			if !exist {
				return fallback
			}
			if hasNext {
				return walk(current, fallback, traversal[1:])
			}
			return variables
		}
		return fallback
	default:
		panic("eval: unsupported traversal: " + reflect.TypeOf(t).String())
	}
}
