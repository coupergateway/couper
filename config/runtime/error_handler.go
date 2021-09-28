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

func newErrorHandler(ctx *hcl.EvalContext, opts *protectedOptions, log *logrus.Entry,
	defs ACDefinitions, references ...string) (http.Handler, error) {
	kindsHandler := map[string]http.Handler{}
	for _, ref := range references {
		for _, h := range defs[ref].ErrorHandler {
			for _, k := range h.Kinds {
				if _, exist := kindsHandler[k]; exist {
					log.Fatal("error type handler exists already: " + k)
				}

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

				epOpts, err := newEndpointOptions(ctx, epConf, nil, opts.srvOpts, log, opts.proxyFromEnv, opts.memStore)
				if err != nil {
					return nil, err
				}
				if epOpts.Error == nil || h.ErrorFile == "" {
					epOpts.Error = opts.epOpts.Error
				}

				epOpts.Error = epOpts.Error.WithContextFunc(func(rw http.ResponseWriter, r *http.Request) {
					beresp := &http.Response{Header: rw.Header()}
					_ = eval.ApplyResponseContext(r.Context(), contextBody, beresp)
				})

				if epOpts.Response != nil && reflect.DeepEqual(epOpts.Response.Context, emptyBody) {
					epOpts.Response = nil
				}

				epOpts.LogHandlerKind = "error_" + k
				kindsHandler[k] = handler.NewEndpoint(epOpts, log, nil)
			}
		}
	}
	return handler.NewErrorHandler(kindsHandler, opts.epOpts.Error), nil
}
