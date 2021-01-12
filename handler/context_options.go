package handler

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/internal/seetie"
)

const (
	attrReqHeaders     = "request_headers"
	attrResHeaders     = "response_headers"
	attrSetReqHeaders  = "set_request_headers"
	attrSetResHeaders  = "set_response_headers"
	attrAddQueryParams = "add_query_params"
	attrDelQueryParams = "remove_query_params"
	attrSetQueryParams = "set_query_params"
)

type OptionsMap map[string][]string

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
