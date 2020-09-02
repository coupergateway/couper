package handler

import (
	"github.com/avenga/couper/internal/seetie"
	"github.com/hashicorp/hcl/v2"
)

type OptionsMap map[string][]string

func NewCtxOptions(attrName string, evalCtx *hcl.EvalContext, body hcl.Body) (OptionsMap, error) {
	var diags hcl.Diagnostics
	var options OptionsMap

	content, d := body.Content(headersAttributeSchema)
	diags = append(diags, d...)

	for _, attr := range content.Attributes {
		if attr.Name != attrName {
			continue
		}
		o, d := NewOptionsMap(evalCtx, attr)
		diags = append(diags, d...)
		options = o
		break
	}

	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}

func NewOptionsMap(evalCtx *hcl.EvalContext, attr *hcl.Attribute) (OptionsMap, hcl.Diagnostics) {
	options := make(OptionsMap)
	expMap, diags := seetie.ExpToMap(evalCtx, attr.Expr)
	if diags.HasErrors() {
		return nil, diags
	}

	for key, val := range expMap {
		switch val.(type) {
		case string:
			options[key] = []string{val.(string)}
			continue
		case []string:
			options[key] = val.([]string)
		}
	}
	return options, nil
}

const (
	attrReqHeaders = "request_headers"
	attrResHeaders = "response_headers"
)

var headersAttributeSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: attrReqHeaders,
		},
		{
			Name: attrResHeaders,
		},
	},
}
