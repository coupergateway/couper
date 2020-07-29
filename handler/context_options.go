package handler

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

func NewCtxOptions(target http.Header, decodeCtx *hcl.EvalContext, body hcl.Body) error {
	var diags hcl.Diagnostics
	content, d := body.Content(headersAttributeSchema)
	diags = append(diags, d...)

	if attr, ok := content.Attributes["request_headers"]; ok {
		emap, mapDiags := hcl.ExprMap(attr.Expr)
		diags = append(diags, mapDiags...)
		for i := range emap {
			val, valDiags := emap[i].Value.Value(decodeCtx)
			diags = append(diags, valDiags...)
			key, keyDiags := emap[i].Key.Value(decodeCtx)
			diags = append(diags, keyDiags...)
			if val.Type().IsPrimitiveType() {
				target.Set(key.AsString(), val.AsString())
				continue
			}
			var values []string
			for _, v := range val.AsValueSlice() {
				values = append(values, v.AsString())
			}
			target[key.AsString()] = values
		}

	}
	if diags.HasErrors() {
		return diags
	}
	return nil
}

var headersAttributeSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "request_headers",
		},
		{
			Name: "response_headers",
		},
	},
}
