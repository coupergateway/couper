package runtime

import (
	"fmt"
	"reflect"
	"sort"
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
	sequences, requests, proxies := newSequence(allProxies, allRequests, errCh, items...)
	if err := <-errCh; err != nil {
		return nil, err
	}

	backendConf := *DefaultBackendConf
	if diags := gohcl.DecodeBody(endpointConf.Remain, confCtx, &backendConf); diags.HasErrors() {
		return nil, diags
	}
	// TODO: redirect
	if endpointConf.Response == nil && len(proxies)+len(requests)+len(sequences) == 0 { // && redirect == nil
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
		APIName:      apiName,
		Context:      endpointConf.Remain,
		Error:        errTpl,
		LogPattern:   endpointConf.Pattern,
		Proxies:      proxies,
		ReqBodyLimit: bodyLimit,
		BufferOpts:   bufferOpts,
		Requests:     requests,
		Sequences:    sequences,
		Response:     response,
		ServerOpts:   serverOptions,
	}, nil
}

// newSequence lookups any request related dependency and sort them into a sequence.
// Also return left-overs for parallel usage.
func newSequence(proxies map[string]*producer.Proxy, requests map[string]*producer.Request, errCh chan<- error,
	items ...config.SequenceItem) (producer.Sequences, producer.Requests, producer.Proxies) {

	defer func() {
		if rc := recover(); rc != nil {
			errCh <- rc.(error)
		}
		close(errCh)
	}()

	// start with default item
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].GetName() == "default"
	})

	var allDeps [][]string
	for _, item := range items {
		deps := make([]string, 0)
		seen := make([]string, 0)
		resolveSequence(item, &deps, &seen)
		allDeps = append(allDeps, deps)
	}

	// part of other chain? filter out
	allDeps = func() (result [][]string) {
	all:
		for _, deps := range allDeps {
			if len(deps) == 1 {
				continue
			}

			for _, otherDeps := range allDeps {
				if len(deps) >= len(otherDeps) {
					continue
				}
				if reflect.DeepEqual(deps, otherDeps[:len(deps)]) {
					continue all
				}
			}
			result = append(result, deps)
		}

		return result
	}()

	var reqs producer.Requests
	var ps producer.Proxies
	var seqs producer.Sequences

	for _, deps := range allDeps {
		var seq producer.Sequence
		for _, dep := range deps {
			if p, ok := proxies[dep]; ok {
				seq = append(seq, &producer.SequenceItem{
					Backend: p.RoundTrip,
					Name:    p.Name,
				})
			}
			if r, ok := requests[dep]; ok {
				seq = append(seq, &producer.SequenceItem{
					Backend: r.Backend,
					Context: r.Context,
					Name:    r.Name,
				})
			}
		}
		seqs = append(seqs, seq)
	}

proxyLeftovers:
	for name, p := range proxies {
		for _, deps := range allDeps {
			for _, dep := range deps {
				if name == dep {
					continue proxyLeftovers
				}
			}
		}
		ps = append(ps, p)
	}

reqLeftovers:
	for name, r := range requests {
		for _, deps := range allDeps {
			for _, dep := range deps {
				if name == dep {
					continue reqLeftovers
				}
			}
		}
		reqs = append(reqs, r)
	}

	return seqs, reqs, ps
}

func resolveSequence(item config.SequenceItem, resolved, seen *[]string) {
	name := item.GetName()
	*seen = append(*seen, name)
	for _, dep := range item.Deps() {
		if !containsString(resolved, dep.GetName()) {
			if !containsString(seen, dep.GetName()) {
				resolveSequence(dep, resolved, seen)
				continue
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
