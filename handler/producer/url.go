package producer

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/utils"
)

func NewURLFromAttribute(hclCtx *hcl.EvalContext, content *hclsyntax.Body, attrName string, req *http.Request) (*url.URL, error) {
	urlVal, err := eval.ValueFromBodyAttribute(hclCtx, content, attrName)
	if err != nil {
		return nil, err
	}

	if urlVal.Type() == cty.NilType { // not set
		return req.URL, nil
	}

	if urlVal.Type() != cty.String {
		return nil, fmt.Errorf("url %q: not a string", attrName)
	}

	urlStr := urlVal.AsString()
	if urlStr == "" { // no attr
		return req.URL, nil
	}

	u, err := url.ParseRequestURI(urlStr) // method forces abs path
	if err != nil {
		return nil, err
	}

	pathMatch, wildcardEP := req.Context().Value(request.Wildcard).(string)

	path := u.Path

	if wildcardEP {
		if strings.HasSuffix(u.Path, "/**") {
			if strings.HasSuffix(req.URL.Path, "/") && !strings.HasSuffix(pathMatch, "/") {
				pathMatch += "/"
			}

			path = utils.JoinPath("/", strings.ReplaceAll(u.Path, "/**", "/"), pathMatch)
		}
	}

	u.Path = path
	return u, nil
}

// removeHost prevents client-request host to leak into backend structure.
func removeHost(outreq *http.Request) {
	outreq.Host = ""
	outreq.URL.Host = ""
}
