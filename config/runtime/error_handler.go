package runtime

import (
	"net/http"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
)

func newErrorHandler(ctx *hcl.EvalContext, conf *config.Couper, opts *protectedOptions, log *logrus.Entry,
	defs ACDefinitions, references ...string) (http.Handler, eval.BufferOption, error) {
	kindsHandler := map[string]http.Handler{}
	var ehBufferOption eval.BufferOption
	for _, ref := range references {
		definition, ok := defs[ref]
		if !ok {
			continue
		}

		handlersPerKind := make(map[string]*config.ErrorHandler)
		for _, h := range definition.ErrorHandler {
			for _, k := range h.Kinds {
				handlersPerKind[k] = h
			}
		}

		if superKindMap, mapExists := errors.SuperTypesMapsByContext[ref]; mapExists {
			// expand super-kinds:
			// * set super-kind error handler for mapped sub-kinds, if no error handler for this sub-kind is already set
			// * remove super-kind error handler for super-kind
			for superKind, subKinds := range superKindMap {
				if skHandler, skExists := handlersPerKind[superKind]; skExists {
					for _, subKind := range subKinds {
						if _, exists := handlersPerKind[subKind]; !exists {
							handlersPerKind[subKind] = skHandler
						}
					}

					delete(handlersPerKind, superKind)
				}
			}
		}

		for k, h := range handlersPerKind {
			contextBody := h.HCLBody()

			epConf := &config.Endpoint{
				Remain:    contextBody,
				Proxies:   h.Proxies,
				ErrorFile: h.ErrorFile,
				Requests:  h.Requests,
				Response:  h.Response,
			}

			emptyBody := &hclsyntax.Body{}
			if epConf.Response == nil { // Set dummy resp to skip related requirement checks, allowed for error_handler.
				epConf.Response = &config.Response{Remain: emptyBody}
			}

			epOpts, err := NewEndpointOptions(ctx, epConf, nil, opts.srvOpts, log, conf, opts.memStore)
			if err != nil {
				return nil, eval.BufferNone, err
			}
			if epOpts.ErrorTemplate == nil || h.ErrorFile == "" {
				epOpts.ErrorTemplate = opts.epOpts.ErrorTemplate
			}

			epOpts.ErrorTemplate = epOpts.ErrorTemplate.WithContextFunc(func(rw http.ResponseWriter, r *http.Request) {
				beresp := &http.Response{Header: rw.Header()}
				_ = eval.ApplyResponseContext(eval.ContextFromRequest(r).HCLContextSync(), contextBody, beresp)
			})

			if epOpts.Response != nil && reflect.DeepEqual(epOpts.Response.Context, emptyBody) {
				epOpts.Response = nil
			}

			epOpts.LogHandlerKind = "error_" + k
			epOpts.IsErrorHandler = true
			kindsHandler[k] = handler.NewEndpoint(epOpts, log, nil)
			ehBufferOption |= epOpts.BufferOpts
		}
	}

	return handler.NewErrorHandler(kindsHandler, opts.epOpts.ErrorTemplate), ehBufferOption, nil
}
