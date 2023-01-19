package configload

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/configload/collect"
	"github.com/avenga/couper/errors"
)

func newCatchAllEndpoint() *config.Endpoint {
	responseBody := hclbody.NewHCLSyntaxBodyWithAttr("status", cty.NumberIntVal(http.StatusNotFound), hcl.Range{})

	return &config.Endpoint{
		Pattern: "/**",
		Remain:  &hclsyntax.Body{},
		Response: &config.Response{
			Remain: responseBody,
		},
	}
}

func refineEndpoints(helper *helper, endpoints config.Endpoints, checkPathPattern bool, definedACs map[string]struct{}) error {
	var err error

	for _, ep := range endpoints {
		if checkPathPattern && ep.Pattern == "" {
			var r *hcl.Range
			if ep.Remain != nil {
				r = getRange(ep.HCLBody())
			}
			return newDiagErr(r, "endpoint: missing path pattern")
		}

		endpointBody := ep.HCLBody()
		if definedACs != nil {
			if err := checkReferencedAccessControls(endpointBody, ep.AccessControl, ep.DisableAccessControl, definedACs); err != nil {
				return err
			}
		}

		rp := endpointBody.Attributes["beta_required_permission"]
		if rp != nil {
			ep.RequiredPermission = rp.Expr
		}

		if checkPathPattern && ep.AllowedMethods != nil && len(ep.AllowedMethods) > 0 {
			if err = validMethods(ep.AllowedMethods, endpointBody.Attributes["allowed_methods"]); err != nil {
				return err
			}
		}

		if checkPathPattern && len(ep.Proxies)+len(ep.Requests) == 0 && ep.Response == nil {
			r := endpointBody.SrcRange
			return newDiagErr(&r,
				"endpoint: missing 'default' proxy or request block, or a response definition",
			)
		}

		proxyRequestLabelRequired := len(ep.Proxies)+len(ep.Requests) > 1

		names := map[string]*hclsyntax.Body{}
		unique := map[string]struct{}{}
		subject := ep.HCLBody().SrcRange

		for _, proxyConfig := range ep.Proxies {
			if proxyConfig.Name == "" {
				proxyConfig.Name = config.DefaultNameLabel
			}

			names[proxyConfig.Name] = proxyConfig.HCLBody()

			subject = proxyConfig.HCLBody().SrcRange
			if err = validLabel(proxyConfig.Name, &subject); err != nil {
				return err
			}

			if proxyRequestLabelRequired {
				if err = uniqueLabelName("proxy and request", unique, proxyConfig.Name, &subject); err != nil {
					return err
				}
			}

			wsEnabled, wsBody, wsErr := getWebsocketsConfig(proxyConfig)
			if wsErr != nil {
				return wsErr
			}

			if wsEnabled {
				if proxyConfig.Name != config.DefaultNameLabel {
					return errors.Configuration.Message("websockets attribute or block is only allowed in a 'default' proxy block")
				}
				if proxyRequestLabelRequired || ep.Response != nil {
					return errors.Configuration.Message("websockets are allowed in the endpoint; other 'proxy', 'request' or 'response' blocks are not allowed")
				}

				if wsBody != nil {
					proxyConfig.Remain = hclbody.MergeBodies(proxyConfig.HCLBody(), wsBody, true)
				}
			}

			proxyConfig.Backend, err = PrepareBackend(helper, "", "", proxyConfig)
			if err != nil {
				return err
			}
		}

		for _, reqConfig := range ep.Requests {
			if reqConfig.Name == "" {
				reqConfig.Name = config.DefaultNameLabel
			}

			names[reqConfig.Name] = reqConfig.HCLBody()

			subject = reqConfig.HCLBody().SrcRange
			if err = validLabel(reqConfig.Name, &subject); err != nil {
				return err
			}

			if proxyRequestLabelRequired {
				if err = uniqueLabelName("proxy and request", unique, reqConfig.Name, &subject); err != nil {
					return err
				}
			}

			// remap request specific names for headers and query to well known ones
			reqBody := reqConfig.HCLBody()
			if err = verifyBodyAttributes(request, reqBody); err != nil {
				return err
			}

			hclbody.RenameAttribute(reqBody, "headers", "set_request_headers")
			hclbody.RenameAttribute(reqBody, "query_params", "set_query_params")

			reqConfig.Backend, err = PrepareBackend(helper, "", "", reqConfig)
			if err != nil {
				return err
			}
		}

		if ep.Response != nil {
			if err = verifyResponseBodyAttrs(ep.Response.HCLBody()); err != nil {
				return err
			}
		}

		if _, ok := names[config.DefaultNameLabel]; checkPathPattern && !ok && ep.Response == nil {
			return newDiagErr(&subject, "Missing a 'default' proxy or request definition, or a response block")
		}

		if err = buildSequences(names, ep); err != nil {
			return err
		}

		epErrorHandler := collect.ErrorHandlerSetters(ep)
		if err = configureErrorHandler(epErrorHandler, helper); err != nil {
			return err
		}
	}

	return nil
}

func getWebsocketsConfig(proxyConfig *config.Proxy) (bool, *hclsyntax.Body, error) {
	hasWebsocketBlocks := len(hclbody.BlocksOfType(proxyConfig.HCLBody(), "websockets")) > 0
	if proxyConfig.Websockets != nil && hasWebsocketBlocks {
		hr := proxyConfig.HCLBody().Attributes["websockets"].SrcRange
		return false, nil, newDiagErr(&hr, "either websockets attribute or block is allowed")
	}

	if proxyConfig.Websockets != nil {
		var body *hclsyntax.Body

		if *proxyConfig.Websockets {
			block := &hclsyntax.Block{
				Type: "websockets",
				Body: &hclsyntax.Body{},
			}

			body = &hclsyntax.Body{Blocks: []*hclsyntax.Block{block}}
		}

		return *proxyConfig.Websockets, body, nil
	}

	return hasWebsocketBlocks, nil, nil
}
