package handler

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

var _ http.Handler = &Error{}

type Error struct {
	kindContext map[string]hcl.Body
	template    *errors.Template
}

func NewErrorHandler(kindContext map[string]hcl.Body, tpl *errors.Template) *Error {
	return &Error{
		kindContext: kindContext,
		template:    tpl,
	}
}

func (e *Error) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	errKind, ok := req.Context().Value(request.ErrorKind).(error)
	if !ok {
		e.template.ServeError(errors.Server).ServeHTTP(rw, req)
		return
	}

	kind := typeToKind(errKind)
	if eh, ok := e.kindContext[kind]; ok {
		resp := newResponse(eh, req)
		eval.ApplyResponseContext(req.Context(), eh, resp)
		resp.Write(rw)
		return
	}

	// TODO: more generic fallback, may fit for access control
	e.template.ServeError(errors.AccessControl).ServeHTTP(rw, req)
}

// copy from endpoint, TODO: refactor and combine
func newResponse(context hcl.Body, req *http.Request) *http.Response {
	clientres := &http.Response{
		Header:     make(http.Header),
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Request:    req,
	}

	hclCtx := req.Context().Value(eval.ContextType).(*eval.Context).HCLContext()

	respContent, _, _ := context.PartialContent(config.ErrorHandler{}.Schema(true))
	resps := respContent.Blocks.OfType("response")
	content := &hcl.BodyContent{}
	if len(resps) > 0 {
		content, _, _ = resps[0].Body.PartialContent(config.ResponseInlineSchema)
	}

	statusCode := http.StatusOK
	if attr, ok := content.Attributes["status"]; ok {
		val, err := attr.Expr.Value(hclCtx)
		if err != nil {
			//e.log.Errorf("endpoint eval error: %v", err)
			statusCode = http.StatusInternalServerError
		} else if statusValue := int(seetie.ValueToInt(val)); statusValue > 0 {
			statusCode = statusValue
		}
	}
	clientres.StatusCode = statusCode
	clientres.Status = http.StatusText(clientres.StatusCode)

	if attr, ok := content.Attributes["headers"]; ok {
		val, _ := attr.Expr.Value(hclCtx)

		eval.SetHeader(val, clientres.Header)
	}

	if attr, ok := content.Attributes["body"]; ok {
		val, _ := attr.Expr.Value(hclCtx)

		r := strings.NewReader(seetie.ValueToString(val))
		clientres.Body = eval.NewReadCloser(r, nil)
	}

	return clientres
}

func typeToKind(err error) string {
	str := err.Error()
	if es, ok := err.(fmt.Stringer); ok {
		str = es.String()
	}
	var result []rune
	for i, c := range str {
		if i == 0 {
			result = append(result, unicode.ToLower(c))
			continue
		}
		if unicode.IsUpper(c) {
			result = append(result, append([]rune{'_'}, unicode.ToLower(c))...)
			continue
		}
		result = append(result, c)
	}
	return string(result)
}
