package runtime

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/config/sequence"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
)

func newEndpointMap(srvConf *config.Server, serverOptions *server.Options) (endpointMap, error) {
	endpoints := make(endpointMap)

	catchAllEndpoints := make(map[string]struct{})
	for _, apiConf := range srvConf.APIs {
		basePath := serverOptions.APIBasePaths[apiConf]
		for _, epConf := range apiConf.Endpoints {
			endpoints[epConf] = apiConf
			if epConf.Pattern == "/**" {
				catchAllEndpoints[basePath] = struct{}{}
			}
		}
	}

	for _, apiConf := range srvConf.APIs {
		basePath := serverOptions.APIBasePaths[apiConf]
		if _, exists := catchAllEndpoints[basePath]; exists {
			continue
		}

		if len(newAC(srvConf, apiConf).List()) == 0 {
			continue
		}

		var (
			spaPaths                          []string
			filesPaths                        []string
			isAPIBasePathUniqueToFilesAndSPAs = true
		)

		if len(serverOptions.SPABasePaths) == 0 {
			spaPaths = []string{""}
		} else {
			spaPaths = serverOptions.SPABasePaths
		}

		if len(serverOptions.FilesBasePaths) == 0 {
			filesPaths = []string{""}
		} else {
			filesPaths = serverOptions.FilesBasePaths
		}

	uniquePaths:
		for _, spaPath := range spaPaths {
			for _, filesPath := range filesPaths {
				isAPIBasePathUniqueToFilesAndSPAs = basePath != filesPath && basePath != spaPath

				if !isAPIBasePathUniqueToFilesAndSPAs {
					break uniquePaths
				}
			}
		}

		if isAPIBasePathUniqueToFilesAndSPAs {
			endpoints[apiConf.CatchAllEndpoint] = apiConf
			catchAllEndpoints[basePath] = struct{}{}
		}
	}

	for _, epConf := range srvConf.Endpoints {
		endpoints[epConf] = nil
	}

	return endpoints, nil
}

func NewEndpointOptions(confCtx *hcl.EvalContext, endpointConf *config.Endpoint, apiConf *config.API,
	serverOptions *server.Options, log *logrus.Entry, conf *config.Couper, memStore *cache.MemoryStore) (*handler.EndpointOptions, error) {
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
			Context: endpointConf.Response.HCLBody(),
		}
		blockBodies = append(blockBodies, response.Context)
	}

	allProxies := make(map[string]*producer.Proxy)
	for _, proxyConf := range endpointConf.Proxies {
		backend, berr := NewBackend(confCtx, proxyConf.Backend, log, conf, memStore)
		if berr != nil {
			return nil, berr
		}
		proxyHandler := handler.NewProxy(backend, proxyConf.HCLBody(), log)
		p := &producer.Proxy{
			Content:   proxyConf.HCLBody(),
			Name:      proxyConf.Name,
			RoundTrip: proxyHandler,
		}

		allProxies[proxyConf.Name] = p
		blockBodies = append(blockBodies, proxyConf.Backend, proxyConf.HCLBody())
	}

	allRequests := make(map[string]*producer.Request)
	for _, requestConf := range endpointConf.Requests {
		backend, berr := NewBackend(confCtx, requestConf.Backend, log, conf, memStore)
		if berr != nil {
			return nil, berr
		}

		pr := &producer.Request{
			Backend: backend,
			Context: requestConf.HCLBody(),
			Name:    requestConf.Name,
		}

		allRequests[requestConf.Name] = pr
		blockBodies = append(blockBodies, requestConf.Backend, requestConf.HCLBody())
	}

	sequences, requests, proxies := resolveDependencies(allProxies, allRequests, endpointConf.Sequences...)

	// TODO: redirect
	if endpointConf.Response == nil && len(proxies)+len(requests)+len(sequences) == 0 { // && redirect == nil
		r := endpointConf.HCLBody().SrcRange
		m := fmt.Sprintf("configuration error: endpoint %q requires at least one proxy, request, response or redirect block", endpointConf.Pattern)
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  m,
			Subject:  &r,
		}}
	}

	bodyLimit, err := parseBodyLimit(endpointConf.RequestBodyLimit)
	if err != nil {
		r := endpointConf.HCLBody().SrcRange
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
		APIName:       apiName,
		Context:       endpointConf.HCLBody(),
		ErrorTemplate: errTpl,
		LogPattern:    endpointConf.Pattern,
		Proxies:       proxies,
		ReqBodyLimit:  bodyLimit,
		BufferOpts:    bufferOpts,
		Requests:      requests,
		Sequences:     sequences,
		Response:      response,
		ServerOpts:    serverOptions,
	}, nil
}

// resolveDependencies lookups any request related dependency and sort them into a sequence.
// Also return left-overs for parallel usage.
func resolveDependencies(proxies map[string]*producer.Proxy, requests map[string]*producer.Request,
	items ...*sequence.Item) (producer.Parallel, producer.Requests, producer.Proxies) {

	allDeps := sequence.Dependencies(items)

	var reqs producer.Requests
	var ps producer.Proxies
	var seqs producer.Parallel
	roundtrips := map[string]producer.Roundtrip{}

	// read from prepared config sequences
	for _, seq := range items {
		seqs = append(seqs, newRoundtrip(seq, roundtrips, proxies, requests))
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

func newRoundtrip(seq *sequence.Item,
	roundtrips map[string]producer.Roundtrip,
	proxies map[string]*producer.Proxy,
	requests map[string]*producer.Request) producer.Roundtrip {

	deps := seq.Deps()
	var rt producer.Roundtrip

	var previous []string
	if len(deps) > 1 { // more deps per item can be parallelized
		var names []string
		for _, d := range deps {
			names = append(names, d.Name)
		}
		k := fmt.Sprintf("%v", names)
		for _, d := range deps {
			previous = append(previous, d.Name)
		}
		if np, ok := roundtrips[k]; ok {
			rt = np
		} else {
			var pl producer.Parallel
			for _, d := range deps {
				pl = append(pl, newRoundtrip(d, roundtrips, proxies, requests))
			}
			rt = &pl
			roundtrips[k] = &pl
		}
	} else if len(deps) == 1 {
		rt = newRoundtrip(deps[0], roundtrips, proxies, requests)
		previous = append(previous, deps[0].Name)
	}

	leaf := newLeafRoundtrip(seq.Name, strings.Join(previous, ","), roundtrips, proxies, requests)
	if rt != nil {
		var names []string
		names = append(names, rt.Names()...)
		names = append(names, leaf.Names()...)
		k := fmt.Sprintf("%v", names)
		if ns, ok := roundtrips[k]; ok {
			return ns
		}
		s := &producer.Sequence{rt, leaf}
		roundtrips[k] = s
		return s
	}
	return leaf
}

// newLeafRoundtrip creates a "leaf" Roundtrip, i.e. one of
// producer.Proxies or producer.Requests,
// no producer.Parallel or producer.Sequence
func newLeafRoundtrip(name, previous string,
	roundtrips map[string]producer.Roundtrip,
	proxies map[string]*producer.Proxy,
	requests map[string]*producer.Request) producer.Roundtrip {
	if rt, ok := roundtrips[name]; ok {
		return rt
	}
	if p, ok := proxies[name]; ok {
		p.PreviousSequence = previous
		ps := &producer.Proxies{p}
		roundtrips[name] = ps
		return ps
	}
	if r, ok := requests[name]; ok {
		r.PreviousSequence = previous
		rs := &producer.Requests{r}
		roundtrips[name] = rs
		return rs
	}
	return nil
}
