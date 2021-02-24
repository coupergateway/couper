package test

import (
	"context"
	"github.com/avenga/couper/handler/transport"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
)

func (h *Helper) NewProxy(conf *transport.Config, backendContext, proxyContext hcl.Body) *handler.Proxy {
	logger, _ := test.NewNullLogger()

	config := conf
	if config == nil {
		config = &transport.Config{
			BackendName:    "HelperUpstream",
			NoProxyFromEnv: true,
		}
	}

	proxyCtx := proxyContext
	if proxyCtx == nil {
		proxyCtx = hcl.EmptyBody()
	}

	evalCtx := eval.NewENVContext(nil)
	backend := transport.NewBackend(evalCtx, backendContext, config, logger.WithContext(context.Background()), nil)

	proxy := handler.NewProxy(backend, proxyCtx, evalCtx)
	return proxy
}

func (h *Helper) NewProxyContext(inlineHCL string) hcl.Body {
	type hclBody struct {
		Inline hcl.Body `hcl:",remain"`
	}

	var remain hclBody
	h.Must(hclsimple.Decode(h.tb.Name()+".hcl", []byte(inlineHCL), eval.NewENVContext(nil), &remain))
	return configload.MergeBodies([]hcl.Body{remain.Inline})
}
