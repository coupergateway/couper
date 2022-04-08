package runtime

import (
	"net/http"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
)

var wildcardMap = map[string][]string{
	"api":      {"beta_insufficient_scope", "beta_operation_denied"},
	"endpoint": {"beta_insufficient_scope", "beta_operation_denied", "sequence", "unexpected_status"},
}

func newErrorHandler(ctx *hcl.EvalContext, opts *protectedOptions, log *logrus.Entry,
	defs ACDefinitions, references ...string) (http.Handler, error) {
	kindsHandler := map[string]http.Handler{}
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

		// this works if wildcard is the only "super kind"
		if mappedKinds, mkExists := wildcardMap[ref]; mkExists {
			// expand wildcard:
			// * set wildcard error handler for mapped kinds, no error handler for this kind is already set
			// * remove wildcard error handler for wildcard
			if wcHandler, wcExists := handlersPerKind["*"]; wcExists {
				for _, mk := range mappedKinds {
					if _, exists := handlersPerKind[mk]; !exists {
						handlersPerKind[mk] = wcHandler
					}
				}
			}
			delete(handlersPerKind, "*")
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

			emptyBody := hcl.EmptyBody()
			if epConf.Response == nil { // Set dummy resp to skip related requirement checks, allowed for error_handler.
				epConf.Response = &config.Response{Remain: emptyBody}
			}

			epOpts, err := newEndpointOptions(ctx, epConf, nil, opts.srvOpts, log, opts.settings, opts.memStore)
			if err != nil {
				return nil, err
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
		}
	}

	return handler.NewErrorHandler(kindsHandler, opts.epOpts.ErrorTemplate), nil
}
