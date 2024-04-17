package validation

import (
	"context"

	"github.com/coupergateway/couper/config/request"
)

type OpenAPIContext struct {
	errors []error
}

func NewWithContext(ctx context.Context) (context.Context, *OpenAPIContext) {
	octx := &OpenAPIContext{}
	return context.WithValue(ctx, request.OpenAPI, octx), octx
}

func (o *OpenAPIContext) Errors() []string {
	if len(o.errors) == 0 {
		return nil
	}
	var result []string
	for _, e := range o.errors {
		result = append(result, e.Error())
	}
	return result
}
