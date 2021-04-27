package producer

import (
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

// Response represents the producer <Response> object.
type Response struct {
	Context hcl.Body
}

func NewResponse(req *http.Request, resp hcl.Body, evalCtx *eval.Context, statusCode int) (*http.Response, error) {
	clientres := &http.Response{
		Header:     make(http.Header),
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Request:    req,
	}

	hclCtx := evalCtx.HCLContext()

	content, _, diags := resp.PartialContent(config.ResponseInlineSchema)
	if diags.HasErrors() {
		return nil, errors.Evaluation.With(diags)
	}

	if attr, ok := content.Attributes["status"]; ok {
		val, err := attr.Expr.Value(hclCtx)
		if err != nil {
			statusCode = http.StatusInternalServerError
		} else if statusValue := int(seetie.ValueToInt64(val)); statusValue > 0 {
			statusCode = statusValue
		}
	}
	clientres.StatusCode = statusCode
	clientres.Status = http.StatusText(clientres.StatusCode)

	respBody, ct, bodyErr := eval.GetBody(hclCtx, content)
	if bodyErr != nil {
		return nil, errors.Evaluation.With(bodyErr)
	}

	if ct != "" {
		clientres.Header.Set("Content-Type", ct)
	}

	if attr, ok := content.Attributes["headers"]; ok {
		val, err := attr.Expr.Value(hclCtx)
		if err != nil {
			return nil, errors.Evaluation.With(err)
		}

		eval.SetHeader(val, clientres.Header)
	}

	if respBody != "" {
		r := strings.NewReader(respBody)
		clientres.Body = io.NopCloser(r)
	}

	return clientres, nil
}
