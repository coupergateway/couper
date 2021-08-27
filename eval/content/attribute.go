package content

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/internal/seetie"
)

type Context interface {
	HCLContext() *hcl.EvalContext
}

func GetContextAttribute(httpContext context.Context, context hcl.Body, name string) (string, error) {
	ctx, ok := httpContext.Value(request.ContextType).(Context)
	if !ok {
		return "", nil
	}
	evalCtx := ctx.HCLContext()

	schema := &hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: name}}}
	content, _, _ := context.PartialContent(schema)
	if content == nil || len(content.Attributes) == 0 {
		return "", nil
	}

	return GetAttribute(evalCtx, content, name)
}

func GetAttribute(ctx *hcl.EvalContext, content *hcl.BodyContent, name string) (string, error) {
	attr := content.Attributes
	if _, ok := attr[name]; !ok {
		return "", nil
	}

	val, diags := attr[name].Expr.Value(ctx)
	if diags.HasErrors() {
		return "", diags
	}

	return seetie.ValueToString(val), nil
}
