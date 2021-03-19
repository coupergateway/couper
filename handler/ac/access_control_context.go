package ac

import (
	"context"

	"github.com/avenga/couper/config/request"
)

type AccessControlContext struct {
	errors []error
}

func NewWithContext(ctx context.Context) (context.Context, *AccessControlContext) {
	octx := &AccessControlContext{}
	return context.WithValue(ctx, request.AccessControl, octx), octx
}

func (o *AccessControlContext) Errors() []string {
	if len(o.errors) == 0 {
		return nil
	}
	var result []string
	for _, e := range o.errors {
		result = append(result, e.Error())
	}
	return result
}
