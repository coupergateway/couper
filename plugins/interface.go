package plugins

import (
	"github.com/avenga/couper/config/schema"
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

// Config defines the given configuration to its parent block.
type Config interface {
	Definition() (parent string, header *hcl.BlockHeaderSchema, schema schema.BodySchema)
}

type HandlerHook interface {
	RegisterHandlerFunc(HookKind, http.Handler)
}

type RoundtripHook interface {
	RegisterRoundtripFunc(HookKind, http.RoundTripper)
}
