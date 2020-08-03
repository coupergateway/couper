package handler

import (
	"math/big"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
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
	var diags hcl.Diagnostics

	emap, mapDiags := hcl.ExprMap(attr.Expr)
	diags = append(diags, mapDiags...)
	for i := range emap {
		val, valDiags := emap[i].Value.Value(evalCtx)
		diags = append(diags, valDiags...)
		key, keyDiags := emap[i].Key.Value(evalCtx)
		diags = append(diags, keyDiags...)
		if key.Type() != cty.String {
			diags = append(diags, &hcl.Diagnostic{
				Context:     &attr.Range,
				Detail:      "key must be a string type",
				EvalContext: evalCtx,
				Expression:  emap[i].Key,
				Severity:    hcl.DiagError,
				Subject:     &attr.Range,
				Summary:     "invalid key type",
			})
			return nil, diags
		}
		if val.Type().IsPrimitiveType() {
			options[key.AsString()] = []string{ValueToString(val)}
			continue
		}
		var values []string
		for _, v := range val.AsValueSlice() {
			str := ValueToString(v)
			if str != "" {
				values = append(values, str)
			}
		}
		options[key.AsString()] = values
	}
	return options, diags
}

var whitespaceRegex = regexp.MustCompile(`^\s*$`)

func ValueToString(v cty.Value) string {
	switch v.Type() {
	case cty.String:
		str := v.AsString()
		if whitespaceRegex.MatchString(str) {
			return ""
		}
		return str
	case cty.Number:
		n := v.AsBigFloat()
		ni, accuracy := n.Int(nil)
		if accuracy == big.Exact {
			return ni.String()
		}
		return n.String()
	default:
		return ""
	}
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
