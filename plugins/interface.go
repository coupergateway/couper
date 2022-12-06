package plugins

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

// Config registers the given configuration to its parent block.
type Config interface {
	Register() (parent string, header *hcl.BlockHeaderSchema, schema *hcl.BodySchema)
}

type HandlerHook interface {
	RegisterHandlerFunc(HookKind, http.Handler)
}

type RoundtripHook interface {
	RegisterRoundtripFunc(HookKind, http.RoundTripper)
}
