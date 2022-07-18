package configload

import (
	"fmt"
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
	responseBody := hclbody.New(&hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{
			"status": {
				Name: "status",
				Expr: &hclsyntax.LiteralValueExpr{
					Val: cty.NumberIntVal(http.StatusNotFound),
				},
			},
		},
	})

	return &config.Endpoint{
		Pattern: "/**",
		Remain:  hclbody.New(&hcl.BodyContent{}),
		Response: &config.Response{
			Remain: responseBody,
		},
	}
}

func refineEndpoints(helper *helper, endpoints config.Endpoints, check bool) error {
	var err error

	for _, endpoint := range endpoints {
		if check && endpoint.Pattern == "" {
			var r *hcl.Range
			if endpoint.Remain != nil {
				r = getRange(endpoint.Remain)
			}
			return newDiagErr(r, "endpoint: missing path pattern")
		}

		endpointContent := bodyToContent(endpoint.Remain)

		rp := endpointContent.Attributes["beta_required_permission"]
		if rp != nil {
			endpoint.RequiredPermission = rp.Expr
		}

		if check && endpoint.AllowedMethods != nil && len(endpoint.AllowedMethods) > 0 {
			if err = validMethods(endpoint.AllowedMethods, &endpointContent.Attributes["allowed_methods"].Range); err != nil {
				return err
			}
		}

		if check && len(endpoint.Proxies)+len(endpoint.Requests) == 0 && endpoint.Response == nil {
			return newDiagErr(&endpointContent.MissingItemRange,
				"missing 'default' proxy or request block, or a response definition",
			)
		}

		proxyRequestLabelRequired := len(endpoint.Proxies)+len(endpoint.Requests) > 1

		names := map[string]hcl.Body{}
		unique := map[string]struct{}{}
		subject := endpoint.Remain.MissingItemRange()

		for _, proxyConfig := range endpoint.Proxies {
			if proxyConfig.Name == "" {
				proxyConfig.Name = defaultNameLabel
			}

			names[proxyConfig.Name] = proxyConfig.Remain

			if err = validLabel(proxyConfig.Name, &subject); err != nil {
				return err
			}

			if proxyRequestLabelRequired {
				if err = uniqueLabelName(unique, proxyConfig.Name, &subject); err != nil {
					return err
				}
			}

			wsEnabled, wsBody, wsErr := getWebsocketsConfig(proxyConfig)
			if wsErr != nil {
				return wsErr
			}

			if wsEnabled {
				if proxyConfig.Name != defaultNameLabel {
					return errors.Configuration.Message("websockets attribute or block is only allowed in a 'default' proxy block")
				}
				if proxyRequestLabelRequired || endpoint.Response != nil {
					return errors.Configuration.Message("websockets are allowed in the endpoint; other 'proxy', 'request' or 'response' blocks are not allowed")
				}

				if wsBody != nil {
					proxyConfig.Remain = hclbody.MergeBodies(proxyConfig.Remain, wsBody)
				}
			}

			proxyConfig.Backend, err = PrepareBackend(helper, "", "", proxyConfig)
			if err != nil {
				return err
			}
		}

		for _, reqConfig := range endpoint.Requests {
			if reqConfig.Name == "" {
				reqConfig.Name = defaultNameLabel
			}

			names[reqConfig.Name] = reqConfig.Remain

			if err = validLabel(reqConfig.Name, &subject); err != nil {
				return err
			}

			if proxyRequestLabelRequired {
				if err = uniqueLabelName(unique, reqConfig.Name, &subject); err != nil {
					return err
				}
			}

			// remap request specific names for headers and query to well known ones
			content, leftOvers, diags := reqConfig.Remain.PartialContent(reqConfig.Schema(true))
			if diags.HasErrors() {
				return diags
			}

			if err := verifyBodyAttributes(request, content); err != nil {
				return err
			}

			renameAttribute(content, "headers", "set_request_headers")
			renameAttribute(content, "query_params", "set_query_params")

			reqConfig.Remain = hclbody.MergeBodies(leftOvers, hclbody.New(content))

			reqConfig.Backend, err = PrepareBackend(helper, "", "", reqConfig)
			if err != nil {
				return err
			}
		}

		if endpoint.Response != nil {
			if err = verifyResponseBodyAttrs(endpoint.Response.HCLBody()); err != nil {
				return err
			}
		}

		if _, ok := names[defaultNameLabel]; check && !ok && endpoint.Response == nil {
			return newDiagErr(&subject, "Missing a 'default' proxy or request definition, or a response block")
		}

		if err = buildSequences(names, endpoint); err != nil {
			return err
		}

		epErrorHandler := collect.ErrorHandlerSetters(endpoint)
		if err = configureErrorHandler(epErrorHandler, helper); err != nil {
			return err
		}
	}

	return nil
}

func getWebsocketsConfig(proxyConfig *config.Proxy) (bool, hcl.Body, error) {
	content, _, diags := proxyConfig.Remain.PartialContent(
		&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "websockets"}}},
	)
	if diags.HasErrors() {
		return false, nil, diags
	}

	if proxyConfig.Websockets != nil && len(content.Blocks.OfType("websockets")) > 0 {
		return false, nil, fmt.Errorf("either websockets attribute or block is allowed")
	}

	if proxyConfig.Websockets != nil {
		var body hcl.Body

		if *proxyConfig.Websockets {
			block := &hcl.Block{
				Type: "websockets",
				Body: hclbody.EmptyBody(),
			}

			body = hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{block}})
		}

		return *proxyConfig.Websockets, body, nil
	}

	return len(content.Blocks) > 0, nil, nil
}

func renameAttribute(content *hcl.BodyContent, old, new string) {
	if attr, ok := content.Attributes[old]; ok {
		attr.Name = new
		content.Attributes[new] = attr
		delete(content.Attributes, old)
	}
}
