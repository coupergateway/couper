package transport

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/health_check"
	"github.com/avenga/couper/handler/validation"
)

// BackendOptions represents the transport <BackendOptions> object.
type BackendOptions struct {
	OpenAPI *validation.OpenAPIOptions

	ParsedOptions *health_check.ParsedOptions
	Request       *http.Request
}

func (bo *BackendOptions) SetRequest(body hcl.Body, evalCtx *hcl.EvalContext) error {
	content, _, _ := body.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{
		{Name: "origin"}},
	})
	origin := cty.BoolVal(false)
	if content != nil {
		if n, exist := content.Attributes["origin"]; exist {
			origin, _ = n.Expr.Value(evalCtx)
		}
	}
	req, err := http.NewRequest(http.MethodGet, origin.AsString(), nil)
	if err != nil {
		return err
	}
	bo.Request = req
	return nil
}
