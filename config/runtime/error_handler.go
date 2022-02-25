package runtime

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
)

func newErrorHandler(ctx *hcl.EvalContext, opts *protectedOptions, log *logrus.Entry,
	defs ACDefinitions, certificate []byte, references ...string) (http.Handler, error) {
	kindsHandler := map[string]http.Handler{}
	for _, ref := range references {
		definition, ok := defs[ref]
		if !ok {
			continue
		}
		for _, h := range definition.ErrorHandler {
			for _, k := range h.Kinds {
				if _, exist := kindsHandler[k]; exist {
					return nil, fmt.Errorf("error handler type already exists: " + k)
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

				epOpts, err := newEndpointOptions(ctx, epConf, nil, opts.srvOpts, log, opts.proxyFromEnv, certificate, opts.memStore)
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
	}

	return handler.NewErrorHandler(kindsHandler, opts.epOpts.ErrorTemplate), nil
}
