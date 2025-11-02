package runtime

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/runtime/server"
	"github.com/coupergateway/couper/config/sequence"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/handler"
	"github.com/coupergateway/couper/handler/producer"
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

	allProducers := make(map[string]producer.Roundtrip)
	for _, proxyConf := range endpointConf.Proxies {
		backend, berr := NewBackend(confCtx, proxyConf.Backend, log, conf, memStore)
		if berr != nil {
			return nil, berr
		}

		var hasWSblock bool
		proxyBody := proxyConf.HCLBody()
		for _, b := range proxyBody.Blocks {
			if b.Type == "websockets" {
				hasWSblock = true
				break
			}
		}

		allowWebsockets := proxyConf.Websockets != nil || hasWSblock
		proxyHandler := handler.NewProxy(backend, proxyBody, allowWebsockets, log)

		p := &producer.Proxy{
			Content:   proxyBody,
			Name:      proxyConf.Name,
			RoundTrip: proxyHandler,
		}

		allProducers[proxyConf.Name] = p
		blockBodies = append(blockBodies, proxyConf.Backend, proxyBody)
	}

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

		allProducers[requestConf.Name] = pr
		blockBodies = append(blockBodies, requestConf.Backend, requestConf.HCLBody())
	}

	markDependencies(allProducers, endpointConf.Sequences)
	addIndependentProducers(allProducers, endpointConf)

	// TODO: redirect
	if endpointConf.Response == nil && len(allProducers) == 0 { // && redirect == nil
		r := endpointConf.HCLBody().SrcRange
		m := fmt.Sprintf("configuration error: endpoint: %q requires at least one proxy, request or response block", endpointConf.Pattern)
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
			Summary:  fmt.Sprintf("endpoint: %q: parsing request body limit", endpointConf.Pattern),
			Subject:  &r,
		}}
	}

	bufferOpts := buffer.Must(append(blockBodies, endpointConf.Remain)...)

	apiName := ""
	if apiConf != nil {
		apiName = apiConf.Name
	}

	return &handler.EndpointOptions{
		APIName:           apiName,
		Context:           endpointConf.HCLBody(),
		ErrorTemplate:     errTpl,
		Items:             endpointConf.Sequences,
		LogPattern:        endpointConf.Pattern,
		Producers:         allProducers,
		ReqBodyLimit:      bodyLimit,
		BufferOpts:        bufferOpts,
		SendServerTimings: conf.Settings.SendServerTimings,
		Response:          response,
		ServerOpts:        serverOptions,
	}, nil
}

func markDependencies(allProducers map[string]producer.Roundtrip, items sequence.List) {
	for _, item := range items {
		pr := allProducers[item.Name]
		var prevs []string
		deps := item.Deps()
		if deps == nil {
			continue
		}
		for _, dep := range deps {
			prevs = append(prevs, dep.Name)
		}
		pr.SetDependsOn(strings.Join(prevs, ","))
		markDependencies(allProducers, deps)
	}
}

func addIndependentProducers(allProducers map[string]producer.Roundtrip, endpointConf *config.Endpoint) {
	// TODO simplify
	allDeps := sequence.Dependencies(endpointConf.Sequences)
	sortedProducers := server.SortDefault(allProducers)
outer:
	for _, name := range sortedProducers {
		for _, deps := range allDeps {
			for _, dep := range deps {
				if name == dep {
					continue outer // in sequence
				}
			}
		}
		endpointConf.Sequences = append(endpointConf.Sequences, &sequence.Item{Name: name})
	}
}
