package test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
)

func (h *Helper) NewProxy(opts *handler.ProxyOptions) (*handler.Proxy, *http.Client, http.Handler, func()) {
	logger, _ := test.NewNullLogger()

	var upstream http.HandlerFunc
	server := httptest.NewServer(upstream)

	proxyOptions := &handler.ProxyOptions{
		Context:     h.NewProxyContext(""),
		BackendName: "HelperUpstream",
		Hostname:    "upstream.helper",
		Origin:      server.URL,
		CORS:        &handler.CORSOptions{},
	}
	proxyOptions = proxyOptions.Merge(opts)

	proxy, err := handler.NewProxy(proxyOptions, logger.WithContext(context.Background()), nil, eval.NewENVContext(nil))
	h.Must(err)

	return proxy.(*handler.Proxy), server.Client(), upstream, server.Close
}

func (h *Helper) NewProxyContext(inlineHCL string) []hcl.Body {
	type hclBody struct {
		Inline hcl.Body `hcl:",remain"`
	}

	var remain hclBody
	h.Must(hclsimple.Decode(h.tb.Name()+".hcl", []byte(inlineHCL), eval.NewENVContext(nil), &remain))
	return []hcl.Body{remain.Inline}
}
