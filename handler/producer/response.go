package producer

import (
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

// Response represents the producer <Response> object.
type Response struct {
	Context *hclsyntax.Body
}

func NewResponse(req *http.Request, resp *hclsyntax.Body, statusCode int) (*http.Response, error) {
	clientres := &http.Response{
		Header:     make(http.Header),
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Request:    req,
	}

	hclCtx := eval.ContextFromRequest(req).HCLContextSync()

	if attr, ok := resp.Attributes["status"]; ok {
		val, err := eval.Value(hclCtx, attr.Expr)
		if err != nil {
			return nil, err
		} else if statusValue := int(seetie.ValueToInt(val)); statusValue > 0 {
			statusCode = statusValue
		}
	}
	clientres.StatusCode = statusCode
	clientres.Status = http.StatusText(clientres.StatusCode)

	respBody, ct, bodyErr := eval.GetBody(hclCtx, resp)
	if bodyErr != nil {
		return nil, errors.Evaluation.With(bodyErr)
	}

	if ct != "" {
		clientres.Header.Set("Content-Type", ct)
	}

	if attr, ok := resp.Attributes["headers"]; ok {
		val, err := eval.Value(hclCtx, attr.Expr)
		if err != nil {
			return nil, err
		}

		eval.SetHeader(val, clientres.Header)
	}

	if respBody != "" {
		r := strings.NewReader(respBody)
		clientres.Body = io.NopCloser(r)
	}

	return clientres, nil
}
