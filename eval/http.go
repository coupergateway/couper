package eval

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

func ApplyRequestContext(ctx *hcl.EvalContext, body hcl.Body, req *http.Request) {
	if req == nil {
		return
	}

}

func ApplyResponseContext(ctx *hcl.EvalContext, body hcl.Body, res *http.Response) {
	if res == nil {
		return
	}
}
