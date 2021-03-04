package producer

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

var (
	_ Roundtrips = Proxies{}
	_ Roundtrips = Requests{}
)

type Roundtrips interface {
	Produce(ctx context.Context, req *http.Request, evalCtx *hcl.EvalContext, results chan<- *Result)
}
