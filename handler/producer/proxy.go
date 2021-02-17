package producer

import (
	"github.com/avenga/couper/handler/transport"
	"github.com/hashicorp/hcl/v2"
)

type Proxy struct {
	Backend *transport.Backend
	Context hcl.Body
}
