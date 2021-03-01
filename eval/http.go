package eval

import (
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

// common "inline" meta-attributes
// TODO: move to config(meta?) package, ask there for req /res related attrs (direction type <> )
const (
	attrPath           = "path"
	attrSetReqHeaders  = "set_request_headers"
	attrAddReqHeaders  = "add_request_headers"
	attrDelReqHeaders  = "remove_request_headers"
	attrAddQueryParams = "add_query_params"
	attrDelQueryParams = "remove_query_params"
	attrSetQueryParams = "set_query_params"

	attrSetResHeaders = "set_response_headers"
	attrAddResHeaders = "add_response_headers"
	attrDelResHeaders = "remove_response_headers"
)

func ApplyRequestContext(ctx *hcl.EvalContext, body hcl.Body, req *http.Request) error {
	if req == nil {
		return nil
	}

	// TODO: bufferOpts from parent
	opts := BufferNone
	httpCtx := NewHTTPContext(ctx, opts, req)

	content, _, _ := body.PartialContent(meta.AttributesSchema)

	headerCtx := req.Header

	// map to name
	// TODO: sorted data structure on load
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range content.Attributes {
		attrs[attr.Name] = attr
	}

	evalURLPath(req, attrs, httpCtx)

	// sort and apply header values in hierarchical and logical order: delete, set, add
	if err := applyHeaderOps(attrs,
		[]string{attrDelReqHeaders, attrSetReqHeaders, attrAddReqHeaders}, httpCtx, headerCtx); err != nil {
		return err
	}

	// Prepare query modifications
	u := *req.URL
	u.RawQuery = strings.ReplaceAll(u.RawQuery, "+", "%2B")
	values := u.Query()
	var modifyQuery bool

	// apply query params in hierarchical and logical order: delete, set, add
	attr, ok := attrs[attrDelQueryParams]
	if ok {
		val, attrDiags := attr.Expr.Value(httpCtx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return attrDiags
		}
		for _, key := range seetie.ValueToStringSlice(val) {
			values.Del(key)
		}
		modifyQuery = true
	}

	attr, ok = attrs[attrSetQueryParams]
	if ok {
		val, attrDiags := attr.Expr.Value(httpCtx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return attrDiags
		}

		for k, v := range seetie.ValueToMap(val) {
			values[k] = toSlice(v)
		}

		modifyQuery = true
	}

	attr, ok = attrs[attrAddQueryParams]
	if ok {
		val, attrDiags := attr.Expr.Value(httpCtx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return attrDiags
		}

		for k, v := range seetie.ValueToMap(val) {
			list := toSlice(v)
			if _, ok = values[k]; !ok {
				values[k] = list
			} else {
				values[k] = append(values[k], list...)
			}
		}

		modifyQuery = true
	}

	if modifyQuery {
		req.URL.RawQuery = strings.ReplaceAll(values.Encode(), "+", "%20")
	}

	return nil
}

func evalURLPath(req *http.Request, attrs map[string]*hcl.Attribute, httpCtx *hcl.EvalContext) {
	path := req.URL.Path
	if pathAttr, ok := attrs[attrPath]; ok {
		pathValue, _ := pathAttr.Expr.Value(httpCtx)
		if str := seetie.ValueToString(pathValue); str != "" {
			path = str
		}
	}

	if pathMatch, ok := req.Context().
		Value(request.Wildcard).(string); ok && strings.HasSuffix(path, "/**") {
		if strings.HasSuffix(req.URL.Path, "/") && !strings.HasSuffix(pathMatch, "/") {
			pathMatch += "/"
		}

		req.URL.Path = utils.JoinPath("/", strings.ReplaceAll(path, "/**", "/"), pathMatch)
	} else if path != "" {
		req.URL.Path = utils.JoinPath("/", path)
	}
}

func ApplyResponseContext(ctx *hcl.EvalContext, body hcl.Body, req *http.Request, beresp *http.Response) error {
	if beresp == nil {
		return nil
	}

	// TODO: bufferOpts from parent
	opts := BufferNone
	httpCtx := NewHTTPContext(ctx, opts, req, beresp)

	content, _, _ := body.PartialContent(meta.AttributesSchema)

	// map to name
	// TODO: sorted data structure on load
	// TODO: func
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range content.Attributes {
		attrs[attr.Name] = attr
	}

	// sort and apply header values in hierarchical and logical order: delete, set, add
	err := applyHeaderOps(attrs,
		[]string{attrDelResHeaders, attrSetResHeaders, attrAddResHeaders}, httpCtx, beresp.Header)
	return err
}

func applyHeaderOps(attrs map[string]*hcl.Attribute, names []string, httpCtx *hcl.EvalContext, headers ...http.Header) error {
	for _, headerCtx := range headers {
		for _, name := range names {
			attr, ok := attrs[name]
			if !ok {
				continue
			}

			val, attrDiags := attr.Expr.Value(httpCtx)
			if seetie.SetSeverityLevel(attrDiags).HasErrors() {
				return attrDiags
			}

			switch name {
			case attrDelReqHeaders, attrDelResHeaders:
				deleteHeader(val, headerCtx)
			case attrSetReqHeaders, attrSetResHeaders:
				SetHeader(val, headerCtx)
			case attrAddReqHeaders, attrAddResHeaders:
				addedHeaders := make(http.Header)
				SetHeader(val, addedHeaders)
				for k, v := range addedHeaders {
					headerCtx[k] = append(headerCtx[k], v...)
				}
			}
		}
	}
	return nil
}

func SetHeader(val cty.Value, headerCtx http.Header) {
	expMap := seetie.ValueToMap(val)
	for key, v := range expMap {
		k := http.CanonicalHeaderKey(key)
		headerCtx[k] = toSlice(v)
	}
}

func deleteHeader(val cty.Value, headerCtx http.Header) {
	for _, key := range seetie.ValueToStringSlice(val) {
		k := http.CanonicalHeaderKey(key)
		headerCtx.Del(k)
	}
}

func toSlice(val interface{}) []string {
	switch val.(type) {
	case string:
		return []string{val.(string)}
	case []string:
		return val.([]string)
	}
	return []string{}
}
