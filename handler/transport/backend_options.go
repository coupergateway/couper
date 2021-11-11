package transport

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/handler/validation"
)

// BackendOptions represents the transport <BackendOptions> object.
type BackendOptions struct {
	OpenAPI     *validation.OpenAPIOptions
	AuthBackend TokenRequest
	HealthCheck *config.HealthCheck
	Request     *http.Request
}

type TokenRequest interface {
	WithToken(req *http.Request) error
	RetryWithToken(req *http.Request, res *http.Response) (bool, error)
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
