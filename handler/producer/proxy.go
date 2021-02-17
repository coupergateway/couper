package producer

import (
	"github.com/avenga/couper/handler/transport"
	"github.com/hashicorp/hcl/v2"
)

// Proxy represents the producer <Proxy> object.
type Proxy struct {
	Backend *transport.Backend
	Context hcl.Body
}

// Proxies represents a list of producer <Proxy> objects.
type Proxies []*Proxy
