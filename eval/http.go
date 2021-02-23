package eval

import (
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/internal/seetie"
)

// common "inline" attribute directives
// TODO: move to config package, ask there for req /res related attrs (direction type <> )
const (
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
	httpCtx := NewHTTPContext(ctx, opts, req, nil, nil)

	attributes, diags := body.JustAttributes()
	if diags.HasErrors() {
		return diags
	}

	headerCtx := req.Header

	// map to name
	// TODO: sorted data structure on load
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range attributes {
		attrs[attr.Name] = attr
	}

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
			return diags
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
			return diags
		}

		for k, v := range seetie.ValueToMap(val) {
			values[k] = v.([]string)
		}

		modifyQuery = true
	}

	attr, ok = attrs[attrAddQueryParams]
	if ok {
		val, attrDiags := attr.Expr.Value(httpCtx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return diags
		}

		for k, v := range seetie.ValueToMap(val) {
			list := v.([]string)
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

func ApplyResponseContext(ctx *hcl.EvalContext, body hcl.Body, req *http.Request, res *http.Response) error {
	if res == nil {
		return nil
	}

	// TODO: bufferOpts from parent
	opts := BufferNone
	httpCtx := NewHTTPContext(ctx, opts, req, res.Request, res)

	attributes, diags := body.JustAttributes()
	if diags.HasErrors() {
		return diags
	}

	// map to name
	// TODO: sorted data structure on load
	// TODO: func
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range attributes {
		attrs[attr.Name] = attr
	}

	// sort and apply header values in hierarchical and logical order: delete, set, add
	err := applyHeaderOps(attrs,
		[]string{attrDelReqHeaders, attrSetReqHeaders, attrAddReqHeaders}, httpCtx, res.Header)
	return err
}

func applyHeaderOps(attrs map[string]*hcl.Attribute, names []string, httpCtx *hcl.EvalContext, headerCtx http.Header) error {
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
			setHeader(val, headerCtx)
		case attrAddReqHeaders, attrAddResHeaders:
			addedHeaders := make(http.Header)
			setHeader(val, addedHeaders)
			for k, v := range addedHeaders {
				headerCtx[k] = append(headerCtx[k], v...)
			}
		}
	}
	return nil
}

func setHeader(val cty.Value, headerCtx http.Header) {
	expMap := seetie.ValueToMap(val)
	for key, v := range expMap {
		k := http.CanonicalHeaderKey(key)
		switch v.(type) {
		case string:
			headerCtx[k] = []string{v.(string)}
			continue
		case []string:
			headerCtx[k] = v.([]string)
		}
	}
}

func deleteHeader(val cty.Value, headerCtx http.Header) {
	for _, key := range seetie.ValueToStringSlice(val) {
		k := http.CanonicalHeaderKey(key)
		if k == "User-Agent" {
			headerCtx[k] = []string{}
			continue
		}
		headerCtx.Del(k)
	}
}
