package eval

import (
	"bytes"
	"context"
	er "errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
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
	attrAddFormParams  = "add_form_params"
	attrDelFormParams  = "remove_form_params"
	attrSetFormParams  = "set_form_params"

	attrSetResHeaders = "set_response_headers"
	attrAddResHeaders = "add_response_headers"
	attrDelResHeaders = "remove_response_headers"
)

// SetGetBody determines if we have to buffer a request body for further processing.
// First of all the user has a related reference within a related options context declaration.
// Additionally the request body is nil or a NoBody type and the http method has no body restrictions like 'TRACE'.
func SetGetBody(req *http.Request, bodyLimit int64) error {
	if req.Method == http.MethodTrace {
		return nil
	}

	// TODO: handle buffer options based on overall body context and reference
	//if (e.opts.ReqBufferOpts & eval.BufferRequest) != eval.BufferRequest {
	//	return nil
	//}

	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		buf := &bytes.Buffer{}
		lr := io.LimitReader(req.Body, bodyLimit+1)
		n, err := buf.ReadFrom(lr)
		if err != nil {
			return err
		}

		if n > bodyLimit {
			return errors.ClientRequest.
				Status(http.StatusRequestEntityTooLarge).
				Message("body size exceeded: " + units.HumanSize(float64(bodyLimit)))
		}

		// reset body initially, additional body reads which are not depending on http.Request
		// internals like form parsing should just call GetBody() and use the returned reader.
		SetBody(req, buf.Bytes())

		// parsing form data now since they read/write request attributes which could be
		// difficult with multiple routines later on.
		parseForm(req)
	}

	return nil
}

// SetBody creates a reader from the given bytes for the Body itself
// and the request GetBody method. Since the size is known the
// Content-Length will be configured too.
func SetBody(req *http.Request, body []byte) {
	bodyBytes := body

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
	}

	req.Body, _ = req.GetBody()

	cl := len(bodyBytes)
	req.Header.Set("Content-Length", strconv.Itoa(cl))
	req.ContentLength = int64(cl)
}

func ApplyRequestContext(ctx context.Context, body hcl.Body, req *http.Request) error {
	if req == nil {
		return nil
	}

	var httpCtx *hcl.EvalContext
	if c, ok := ctx.Value(ContextType).(*Context); ok {
		httpCtx = c.HCLContext()
	}

	content, _, diags := body.PartialContent(meta.AttributesSchema)
	if diags.HasErrors() {
		return diags
	}

	headerCtx := req.Header

	// map to name
	// TODO: sorted data structure on load
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range content.Attributes {
		attrs[attr.Name] = attr
	}

	if err := evalURLPath(req, attrs, httpCtx); err != nil {
		return err
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

	return getFormParams(httpCtx, req, attrs)
}

func getFormParams(ctx *hcl.EvalContext, req *http.Request, attrs map[string]*hcl.Attribute) error {
	const contentTypeValue = "application/x-www-form-urlencoded"

	attrDel, okDel := attrs[attrDelFormParams]
	attrSet, okSet := attrs[attrSetFormParams]
	attrAdd, okAdd := attrs[attrAddFormParams]

	if !okAdd && !okDel && !okSet {
		return nil
	}

	log := req.Context().Value(request.LogEntry).(*logrus.Entry)

	if req.Method != http.MethodPost {
		log.WithError(errors.Evaluation.Label("form_params").
			Messagef("method mismatch: %s", req.Method)).Warn()
		return nil
	}
	if ct := req.Header.Get("Content-Type"); !strings.HasPrefix(strings.ToLower(ct), contentTypeValue) {
		log.WithError(errors.Evaluation.Label("form_params").
			Messagef("content-type mismatch: %s", ct)).Warn()
		return nil
	}

	values := req.PostForm
	if values == nil {
		values = make(url.Values)
	}

	if okDel {
		val, attrDiags := attrDel.Expr.Value(ctx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return attrDiags
		}
		for _, key := range seetie.ValueToStringSlice(val) {
			values.Del(key)
		}
	}

	if okSet {
		val, attrDiags := attrSet.Expr.Value(ctx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return attrDiags
		}

		for k, v := range seetie.ValueToMap(val) {
			values[k] = toSlice(v)
		}
	}

	if okAdd {
		val, attrDiags := attrAdd.Expr.Value(ctx)
		if seetie.SetSeverityLevel(attrDiags).HasErrors() {
			return attrDiags
		}

		for k, v := range seetie.ValueToMap(val) {
			list := toSlice(v)
			if _, okAdd = values[k]; !okAdd {
				values[k] = list
			} else {
				values[k] = append(values[k], list...)
			}
		}
	}

	SetBody(req, []byte(values.Encode()))

	return nil
}

func evalURLPath(req *http.Request, attrs map[string]*hcl.Attribute, httpCtx *hcl.EvalContext) error {
	path := req.URL.Path
	if pathAttr, ok := attrs[attrPath]; ok {
		pathValue, _ := pathAttr.Expr.Value(httpCtx)
		if str := seetie.ValueToString(pathValue); str != "" {
			// TODO: Check for a valid absolute path
			if i := strings.Index(str, "#"); i >= 0 {
				return errors.Configuration.Messagef("path attribute: invalid fragment found in \"%s\"", str)
			} else if i := strings.Index(str, "?"); i >= 0 {
				return errors.Configuration.Messagef("path attribute: invalid query string found in \"%s\"", str)
			}

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

	return nil
}

func ApplyResponseContext(ctx context.Context, body hcl.Body, beresp *http.Response) error {
	if beresp == nil {
		return nil
	}

	var httpCtx *hcl.EvalContext
	if c, ok := ctx.Value(ContextType).(*Context); ok {
		httpCtx = c.eval
	}
	content, _, _ := body.PartialContent(meta.AttributesSchema)

	// map to name
	// TODO: sorted data structure on load
	// TODO: func
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range content.Attributes {
		attrs[attr.Name] = attr
	}

	// sort and apply header values in hierarchical and logical order: delete, set, add
	headers := []string{attrDelResHeaders, attrSetResHeaders, attrAddResHeaders}
	err := applyHeaderOps(attrs, headers, httpCtx, beresp.Header)
	if err != nil {
		return errors.Evaluation.With(err)
	}
	return nil
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

func GetContextAttribute(context hcl.Body, httpContext context.Context, name string) (string, error) {
	ctx, ok := httpContext.Value(ContextType).(*Context)
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

func GetBody(ctx *hcl.EvalContext, content *hcl.BodyContent) (string, string, error) {
	attr, ok := content.Attributes["json_body"]
	if ok {
		val, err := attr.Expr.Value(ctx)
		if err != nil {
			return "", "", err
		}

		val, err1 := stdlib.JSONEncodeFunc.Call([]cty.Value{val})
		if err1 != nil {
			return "", "", err1
		}

		return val.AsString(), "application/json", nil
	}

	attr, ok = content.Attributes["form_body"]
	if ok {
		val, err := attr.Expr.Value(ctx)
		if err != nil {
			return "", "", err
		}

		if valType := val.Type(); !(valType.IsObjectType() || valType.IsMapType()) {
			return "", "", er.New("value of form_body must be object")
		}

		data := url.Values{}
		for k, v := range val.AsValueMap() {
			for _, sv := range seetie.ValueToStringSlice(v) {
				data.Add(k, sv)
			}
		}

		return data.Encode(), "application/x-www-form-urlencoded", nil
	}

	attr, ok = content.Attributes["body"]
	if ok {
		val, err := attr.Expr.Value(ctx)
		if err != nil {
			return "", "", err
		}

		return seetie.ValueToString(val), "text/plain", nil
	}

	return "", "", nil
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
	switch v := val.(type) {
	case float64:
		return []string{strconv.FormatFloat(v, 'f', 0, 64)}
	case string:
		return []string{v}
	case []string:
		return v
	}
	return []string{}
}
