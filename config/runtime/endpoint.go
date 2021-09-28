package runtime

import (
	"fmt"

	"github.com/avenga/couper/cache"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

func newEndpointMap(srvConf *config.Server, serverOptions *server.Options) (endpointMap, error) {
	endpoints := make(endpointMap)

	apiBasePaths := make(map[string]struct{})

	for _, apiConf := range srvConf.APIs {
		basePath := serverOptions.APIBasePaths[apiConf]

		var filesBasePath, spaBasePath string
		if serverOptions.FilesBasePath != "" {
			filesBasePath = serverOptions.FilesBasePath
		}
		if serverOptions.SPABasePath != "" {
			spaBasePath = serverOptions.SPABasePath
		}

		isAPIBasePathUniqueToFilesAndSPA := basePath != filesBasePath && basePath != spaBasePath

		if _, ok := apiBasePaths[basePath]; ok {
			return nil, fmt.Errorf("API paths must be unique")
		}

		apiBasePaths[basePath] = struct{}{}

		for _, epConf := range apiConf.Endpoints {
			endpoints[epConf] = apiConf

			if epConf.Pattern == "/**" {
				isAPIBasePathUniqueToFilesAndSPA = false
			}
		}

		if isAPIBasePathUniqueToFilesAndSPA && len(newAC(srvConf, apiConf).List()) > 0 {
			endpoints[apiConf.CatchAllEndpoint] = apiConf
		}
	}

	for _, epConf := range srvConf.Endpoints {
		endpoints[epConf] = nil
	}

	return endpoints, nil
}

func newEndpointOptions(confCtx *hcl.EvalContext, endpointConf *config.Endpoint, apiConf *config.API,
	serverOptions *server.Options, log *logrus.Entry, proxyEnv bool, memStore *cache.MemoryStore) (*handler.EndpointOptions, error) {
	var errTpl *errors.Template

	if endpointConf.ErrorFile != "" {
		tpl, err := errors.NewTemplateFromFile(endpointConf.ErrorFile, log)
		if err != nil {
			return nil, err
		}
		errTpl = tpl
	} else if apiConf != nil {
		errTpl = serverOptions.APIErrTpls[apiConf]
	} else {
		errTpl = serverOptions.ServerErrTpl
	}

	var response *producer.Response
	// var redirect producer.Redirect // TODO: configure redirect block
	proxies := make(producer.Proxies, 0)
	requests := make(producer.Requests, 0)

	if endpointConf.Response != nil {
		response = &producer.Response{
			Context: endpointConf.Response.Remain,
		}
	}

	for _, proxyConf := range endpointConf.Proxies {
		backend, berr := newBackend(confCtx, proxyConf.Backend, log, proxyEnv, memStore)
		if berr != nil {
			return nil, berr
		}
		proxyHandler := handler.NewProxy(backend, proxyConf.HCLBody(), log)
		p := &producer.Proxy{
			Name:      proxyConf.Name,
			RoundTrip: proxyHandler,
		}
		proxies = append(proxies, p)
	}

	for _, requestConf := range endpointConf.Requests {
		backend, berr := newBackend(confCtx, requestConf.Backend, log, proxyEnv, memStore)
		if berr != nil {
			return nil, berr
		}

		requests = append(requests, &producer.Request{
			Backend: backend,
			Context: requestConf.Remain,
			Name:    requestConf.Name,
		})
	}

	backendConf := *DefaultBackendConf
	if diags := gohcl.DecodeBody(endpointConf.Remain, confCtx, &backendConf); diags.HasErrors() {
		return nil, diags
	}
	// TODO: redirect
	if endpointConf.Response == nil && len(proxies)+len(requests) == 0 { // && redirect == nil
		r := endpointConf.Remain.MissingItemRange()
		m := fmt.Sprintf("configuration error: endpoint %q requires at least one proxy, request, response or redirect block", endpointConf.Pattern)
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  m,
			Subject:  &r,
		}}
	}

	bodyLimit, err := parseBodyLimit(endpointConf.RequestBodyLimit)
	if err != nil {
		r := endpointConf.Remain.MissingItemRange()
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "parsing endpoint request body limit: " + endpointConf.Pattern,
			Subject:  &r,
		}}
	}

	// TODO: determine request/backend_responses.*.body access in this context (all including backend) or for now:
	bufferOpts := eval.MustBuffer(endpointConf.Remain)
	if len(proxies)+len(requests) > 1 { // also buffer with more possible results
		bufferOpts |= eval.BufferResponse
	}

	return &handler.EndpointOptions{
		Context:       endpointConf.Remain,
		Error:         errTpl,
		LogPattern:    endpointConf.Pattern,
		Proxies:       proxies,
		ReqBodyLimit:  bodyLimit,
		ReqBufferOpts: bufferOpts,
		Requests:      requests,
		Response:      response,
		ServerOpts:    serverOptions,
	}, nil
}
