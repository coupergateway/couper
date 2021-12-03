package runtime

import (
	"fmt"
	"strings"

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

	// blockBodies contains inner endpoint block remain bodies to determine req/res buffer options.
	var blockBodies []hcl.Body

	var response *producer.Response
	// var redirect producer.Redirect // TODO: configure redirect block
	proxies := make(producer.Proxies, 0)
	//requests := make(producer.Requests, 0)
	//requestSequence := make(producer.Sequence, 0)

	if endpointConf.Response != nil {
		response = &producer.Response{
			Context: endpointConf.Response.Remain,
		}
		blockBodies = append(blockBodies, response.Context)
	}

	var items []config.SequenceItem
	allProxies := make(map[string]*producer.Proxy)
	for _, proxyConf := range endpointConf.Proxies {
		backend, innerBody, berr := newBackend(confCtx, proxyConf.Backend, log, proxyEnv, memStore)
		if berr != nil {
			return nil, berr
		}
		proxyHandler := handler.NewProxy(backend, proxyConf.HCLBody(), log)
		p := &producer.Proxy{
			Name:      proxyConf.Name,
			RoundTrip: proxyHandler,
		}
		proxies = append(proxies, p)
		allProxies[proxyConf.Name] = p
		items = append(items, proxyConf)
		blockBodies = append(blockBodies, proxyConf.Backend, innerBody, proxyConf.HCLBody())
	}

	allRequests := make(map[string]*producer.Request)
	for _, requestConf := range endpointConf.Requests {
		backend, innerBody, berr := newBackend(confCtx, requestConf.Backend, log, proxyEnv, memStore)
		if berr != nil {
			return nil, berr
		}

		pr := &producer.Request{
			Backend: backend,
			Context: requestConf.Remain,
			Name:    requestConf.Name,
		}

		allRequests[requestConf.Name] = pr
		items = append(items, requestConf)
		blockBodies = append(blockBodies, requestConf.Backend, innerBody, requestConf.HCLBody())
	}

	errCh := make(chan error, 1)
	requestSequence, requests, _ := newSequence(allProxies, allRequests, errCh, items...)
	if err := <-errCh; err != nil {
		return nil, err
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

	bufferOpts := eval.MustBuffer(append(blockBodies, endpointConf.Remain)...)

	apiName := ""
	if apiConf != nil {
		apiName = apiConf.Name
	}

	return &handler.EndpointOptions{
		APIName:         apiName,
		Context:         endpointConf.Remain,
		Error:           errTpl,
		LogPattern:      endpointConf.Pattern,
		Proxies:         proxies,
		ReqBodyLimit:    bodyLimit,
		BufferOpts:      bufferOpts,
		Requests:        requests,
		RequestSequence: requestSequence,
		Response:        response,
		ServerOpts:      serverOptions,
	}, nil
}

// newSequence lookups any request related dependency and sort them into a sequence.
// Also return left-overs for parallel usage.
func newSequence(proxies map[string]*producer.Proxy, requests map[string]*producer.Request, errCh chan<- error,
	items ...config.SequenceItem) (producer.Sequence, producer.Requests, producer.Proxies) {

	defer func() {
		if rc := recover(); rc != nil {
			errCh <- rc.(error)
		}
		close(errCh)
	}()

	deps, seen := make([]string, 0), make([]string, 0)
	for _, item := range items {
		resolveSequence(item, &deps, &seen)
	}

	var reqs producer.Requests
	var ps producer.Proxies
	var seq producer.Sequence

	for _, dep := range deps {
		//if p, ok := proxies[dep]; ok {
		//	seq = append(p)
		//}
		if r, ok := requests[dep]; ok {
			seq = append(seq, r)
		}
	}

leftovers:
	for name, r := range requests {
		for _, dep := range deps {
			if name == dep {
				continue leftovers
			}
		}
		reqs = append(reqs, r)
	}

	return seq, reqs, ps
}

func resolveSequence(item config.SequenceItem, resolved, seen *[]string) {
	name := item.GetName()
	*seen = append(*seen, name)
	for _, dep := range item.Deps() {
		if !containsString(resolved, dep.GetName()) {
			if !containsString(seen, dep.GetName()) {
				resolveSequence(dep, resolved, seen)
			}

			r := hcl.Range{} // try to obtain some config context
			if b, ok := item.(interface{ HCLBody() hcl.Body }); ok {
				r = b.HCLBody().MissingItemRange()
			}
			err := &hcl.Diagnostic{
				Detail:   "circular sequence reference: " + strings.Join(*resolved, ",") + ": " + dep.GetName(),
				Severity: hcl.DiagError,
				Subject:  &r,
				Summary:  "configuration error",
			}
			panic(err)

		}
	}

	*resolved = append(*resolved, name)
}

func containsString(slice *[]string, needle string) bool {
	for _, n := range *slice {
		if n == needle {
			return true
		}
	}
	return false
}
