package ac

import (
	"context"

	"github.com/avenga/couper/config/request"
)

type AccessControlContext struct {
	error error
	name  string
}

func NewWithContext(ctx context.Context) (context.Context, *AccessControlContext) {
	octx := &AccessControlContext{}
	return context.WithValue(ctx, request.AccessControl, octx), octx
}

func (o *AccessControlContext) Name() string {
	return o.name
}

func (o *AccessControlContext) Error() string {
	if o.error == nil {
		return ""
	}

	return o.error.Error()
}
