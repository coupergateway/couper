package plugins

import (
	"context"
	"github.com/avenga/couper/config/schema"
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

type MountPoint string

const (
	Definitions MountPoint = "definitions"
	Endpoint    MountPoint = "endpoint"
)

// Config defines the given configuration to its parent block.
type Config interface {
	Definition() (parent MountPoint, header *hcl.BlockHeaderSchema, schema schema.BodySchema)
	Validate(ctx *hcl.EvalContext, body hcl.Body)
}

type HandlerHook interface {
	RegisterHandlerFunc(HookKind, http.Handler)
}

type ConnectionHook interface {
	Connect(ctx context.Context, args ...interface{})
}
