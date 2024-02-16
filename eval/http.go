package eval

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/coupergateway/couper/config/meta"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval/attributes"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/eval/variables"
	"github.com/coupergateway/couper/internal/seetie"
	"github.com/coupergateway/couper/utils"
)

// SetGetBody determines if we have to buffer a request body for further processing.
// First the user has a related reference within a related options' context declaration.
// Additionally, the request body is nil or a NoBody-Type and the http method has no
// http-body restrictions like 'TRACE'.
func SetGetBody(req *http.Request, bufferOpts buffer.Option, bodyLimit int64) error {
	if req.Method == http.MethodTrace {
		return nil
	}

	if !bufferOpts.Request() {
		return nil
	}

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

		// reset body initially, additional body reads which do not depend on http.Request
		// internals like form parsing should just call GetBody() and use the returned reader.
		SetBody(req, buf.Bytes())
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

	// parsing form data now since they read/write request attributes which could be
	// difficult with multiple routines later on.
	parseForm(req)
}

func ApplyRequestContext(httpCtx *hcl.EvalContext, body *hclsyntax.Body, req *http.Request) error {
	if req == nil {
		return nil
	}

	headerCtx := req.Header

	pathAttr := body.Attributes[variables.Path]
	if pathAttr != nil {
		if err := evalPathAttr(req, pathAttr, httpCtx); err != nil {
			return err
		}
	}

	attrs, err := getAllAttributes(body)
	if err != nil {
		return err
	}

	// sort and apply header values in hierarchical and logical order: delete, set, add
	if err = applyHeaderOps(attrs,
		[]string{attributes.DelReqHeaders, attributes.SetReqHeaders, attributes.AddReqHeaders}, httpCtx, headerCtx); err != nil {
		return err
	}

	// Prepare query modifications
	u := *req.URL
	u.RawQuery = strings.ReplaceAll(u.RawQuery, "+", "%2B")
	values := u.Query()
	var modifyQuery bool

	// apply query params in hierarchical and logical order: delete, set, add
	attr, ok := attrs[attributes.DelQueryParams]
	if ok {
		val, attrErr := Value(httpCtx, attr.Expr)
		if attrErr != nil {
			return attrErr
		}
		for _, key := range seetie.ValueToStringSlice(val) {
			values.Del(key)
		}
		modifyQuery = true
	}

	attr, ok = attrs[attributes.SetQueryParams]
	if ok {
		val, attrErr := Value(httpCtx, attr.Expr)
		if attrErr != nil {
			return attrErr
		}

		for k, v := range seetie.ValueToMap(val) {
			if v == nil {
				continue
			}
			values[k] = toSlice(v)
		}

		modifyQuery = true
	}

	attr, ok = attrs[attributes.AddQueryParams]
	if ok {
		val, attrErr := Value(httpCtx, attr.Expr)
		if attrErr != nil {
			return attrErr
		}

		for k, v := range seetie.ValueToMap(val) {
			if v == nil {
				continue
			}
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

	attrDel, okDel := attrs[attributes.DelFormParams]
	attrSet, okSet := attrs[attributes.SetFormParams]
	attrAdd, okAdd := attrs[attributes.AddFormParams]

	if !okAdd && !okDel && !okSet {
		return nil
	}

	log := req.Context().Value(request.LogEntry).(*logrus.Entry).WithContext(req.Context())

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
		val, attrErr := Value(ctx, attrDel.Expr)
		if attrErr != nil {
			return attrErr
		}
		for _, key := range seetie.ValueToStringSlice(val) {
			values.Del(key)
		}
	}

	if okSet {
		val, attrErr := Value(ctx, attrSet.Expr)
		if attrErr != nil {
			return attrErr
		}

		for k, v := range seetie.ValueToMap(val) {
			if v == nil {
				continue
			}
			values[k] = toSlice(v)
		}
	}

	if okAdd {
		val, attrErr := Value(ctx, attrAdd.Expr)
		if attrErr != nil {
			return attrErr
		}

		for k, v := range seetie.ValueToMap(val) {
			if v == nil {
				continue
			}
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

func evalPathAttr(req *http.Request, pathAttr *hclsyntax.Attribute, httpCtx *hcl.EvalContext) error {
	if pathAttr == nil {
		return nil
	}

	pathValue, err := Value(httpCtx, pathAttr.Expr)
	if err != nil {
		return err
	}

	if path := seetie.ValueToString(pathValue); path != "" {
		// TODO: Check for a valid absolute path
		if i := strings.Index(path, "#"); i >= 0 {
			return errors.Configuration.Label("path attribute").Messagef("invalid fragment found in %q", path)
		} else if i = strings.Index(path, "?"); i >= 0 {
			return errors.Configuration.Label("path attribute").Messagef("invalid query string found in %q", path)
		}

		if pathMatch, isWildcard := req.Context().
			Value(request.Wildcard).(string); isWildcard && strings.HasSuffix(path, "/**") {
			if strings.HasSuffix(req.URL.Path, "/") && !strings.HasSuffix(pathMatch, "/") {
				pathMatch += "/"
			}

			req.URL.Path = utils.JoinPath("/", strings.ReplaceAll(path, "/**", "/"), pathMatch)
		} else if path != "" {
			req.URL.Path = utils.JoinPath("/", path)
		}
	}

	return nil
}

func upgradeType(h http.Header) string {
	return strings.ToLower(h.Get("Upgrade"))
}

func IsUpgradeRequest(req *http.Request) bool {
	if req == nil {
		return false
	}
	if _, ok := req.Context().Value(request.WebsocketsAllowed).(bool); !ok {
		return false
	}
	if conn := strings.ToLower(req.Header.Get("Connection")); !strings.Contains(conn, "upgrade") {
		return false
	}
	if reqUpType := upgradeType(req.Header); reqUpType != "websocket" {
		return false
	}

	return true
}

func IsUpgradeResponse(req *http.Request, res *http.Response) bool {
	if !IsUpgradeRequest(req) || res == nil {
		return false
	}

	return upgradeType(req.Header) == upgradeType(res.Header)
}

var customLogFieldsSchema = &hcl.BodySchema{Attributes: []hcl.AttributeSchema{
	{Name: attributes.CustomLogFields},
}}

func EvalCustomLogFields(httpCtx *hcl.EvalContext, body *hclsyntax.Body) (cty.Value, error) {
	attr, ok := body.Attributes[attributes.CustomLogFields]
	if !ok {
		return cty.NilVal, nil
	}

	return Value(httpCtx, attr.Expr)
}

func ApplyCustomLogs(httpCtx *hcl.EvalContext, bodies []hcl.Body, logger *logrus.Entry) logrus.Fields {
	var values []cty.Value

	for _, body := range bodies {
		if body == nil {
			continue // Test cases
		}

		bodyContent, _, _ := body.PartialContent(customLogFieldsSchema)

		logs, ok := bodyContent.Attributes[attributes.CustomLogFields]
		if !ok {
			continue
		}

		val, err := Value(httpCtx, logs.Expr)
		if err != nil {
			logger.Debug(err)
			continue
		}

		values = append(values, val)
	}

	val, err := lib.Merge(values)
	if err != nil {
		logger.Debug(err)
	}

	return seetie.ValueToLogFields(val)
}

func ApplyResponseContext(ctx *hcl.EvalContext, body *hclsyntax.Body, beresp *http.Response) error {
	if beresp == nil {
		return nil
	}

	if err := ApplyResponseHeaderOps(ctx, body, beresp.Header); err != nil {
		return err
	}

	if IsUpgradeResponse(beresp.Request, beresp) {
		return nil
	}

	if attr, ok := body.Attributes["set_response_status"]; ok {
		_, err := ApplyResponseStatus(ctx, attr, beresp)
		return err
	}

	return nil
}

func ApplyResponseStatus(httpCtx *hcl.EvalContext, attr *hclsyntax.Attribute, beresp *http.Response) (int, error) {
	statusValue, err := Value(httpCtx, attr.Expr)
	if err != nil {
		return 0, err
	}

	status := seetie.ValueToInt(statusValue)
	if status < 100 || status > 599 {
		return 0, errors.Configuration.Label("set_response_status").
			Messagef("invalid http status code: %d", status)
	}

	if beresp != nil {
		if status == 204 {
			beresp.Request.Context().
				Value(request.LogEntry).(*logrus.Entry).
				Warn("set_response_status: removing body, if any due to status-code 204")

			beresp.Body = io.NopCloser(bytes.NewBuffer([]byte{}))
			beresp.ContentLength = -1
			beresp.Header.Del("Content-Length")
		}

		beresp.StatusCode = int(status)
	}

	return int(status), nil
}

func ApplyResponseHeaderOps(httpCtx *hcl.EvalContext, body hcl.Body, headers ...http.Header) error {
	attrs, err := getAllAttributes(body)
	if err != nil {
		return err
	}

	// sort and apply header values in hierarchical and logical order: delete, set, add
	h := []string{attributes.DelResHeaders, attributes.SetResHeaders, attributes.AddResHeaders}
	err = applyHeaderOps(attrs, h, httpCtx, headers...)
	return err
}

func getAllAttributes(body hcl.Body) (map[string]*hcl.Attribute, error) {
	bodyContent, _, diags := body.PartialContent(meta.ModifierAttributesSchema)
	if diags.HasErrors() {
		return nil, errors.Evaluation.With(diags)
	}

	// map to name
	// TODO: sorted data structure on load
	// TODO: func
	attrs := make(map[string]*hcl.Attribute)
	for _, attr := range bodyContent.Attributes {
		attrs[attr.Name] = attr
	}

	return attrs, nil
}

func applyHeaderOps(attrs map[string]*hcl.Attribute, names []string, httpCtx *hcl.EvalContext, headers ...http.Header) error {
	for _, headerCtx := range headers {
		for _, name := range names {
			attr, ok := attrs[name]
			if !ok {
				continue
			}

			val, attrErr := Value(httpCtx, attr.Expr)
			if attrErr != nil {
				return attrErr
			}

			switch name {
			case attributes.DelReqHeaders, attributes.DelResHeaders:
				deleteHeader(val, headerCtx)
			case attributes.SetReqHeaders, attributes.SetResHeaders:
				SetHeader(val, headerCtx)
			case attributes.AddReqHeaders, attributes.AddResHeaders:
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

func GetBody(ctx *hcl.EvalContext, content *hclsyntax.Body) (string, string, error) {
	attr, ok := content.Attributes["json_body"]
	if ok {
		val, err := Value(ctx, attr.Expr)
		if err != nil {
			return "", "", err
		}

		val, err = stdlib.JSONEncodeFunc.Call([]cty.Value{val})
		if err != nil {
			return "", "", errors.Server.With(err)
		}

		return val.AsString(), "application/json", nil
	}

	attr, ok = content.Attributes["form_body"]
	if ok {
		val, err := Value(ctx, attr.Expr)
		if err != nil {
			return "", "", err
		}

		if valType := val.Type(); !(valType.IsObjectType() || valType.IsMapType()) {
			return "", "", errors.Configuration.Message("value of form_body must be an object")
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
		val, err := Value(ctx, attr.Expr)
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
		if v == nil {
			continue
		}
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
	// as this is called with values from a map returned by seetie.ValueToMap(), val can currently be one of
	// * nil
	// * bool
	// * string
	// * float64
	// * []interface{}
	// * map[string]interface{}
	switch v := val.(type) {
	case float64:
		return []string{strconv.FormatFloat(v, 'f', 0, 64)}
	case string:
		return []string{v}
	case []interface{}:
		var l []string
		for _, e := range v {
			s := toString(e)
			if s == nil {
				continue
			}
			l = append(l, *s)
		}
		return l
	}
	return []string{}
}

func toString(val interface{}) *string {
	switch v := val.(type) {
	case string:
		return &v
	case float64:
		s := strconv.FormatFloat(v, 'f', 0, 64)
		return &s
	}
	return nil
}
