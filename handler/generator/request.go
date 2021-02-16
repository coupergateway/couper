package generator

import (
	"io"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/handler/transport"
)

type Request struct {
	Backend *transport.Backend
	Body    io.Reader
	Context hcl.Body
	// Dispatch bool
	Method string
	Name   string // label
	URL    string
}
