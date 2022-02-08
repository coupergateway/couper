package configload

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
)

var backendBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       backend,
			LabelNames: []string{"name"},
		},
	},
}

var defaultBackend = hclbody.New(&hcl.BodyContent{
	Attributes: map[string]*hcl.Attribute{
		"connect_timeout": {
			Name: "connect_timeout",
			Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("10s")},
		},
		"ttfb_timeout": {
			Name: "ttfb_timeout",
			Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("60s")},
		},
		"timeout": {
			Name: "timeout",
			Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("300s")},
		},
	},
})

func PrepareBackend(helper *Helper, attrName, attrValue string, block config.Inline) (hcl.Body, error) {
	var reference string // backend definitions
	var backendBody hcl.Body
	var err error

	reference, backendBody, err = getBackendBody(block)
	if err != nil {
		return nil, err
	}

	if beref, ok := block.(config.BackendReference); ok {
		if beref.Reference() != "" {
			reference = beref.Reference()
		}
	}

	if reference != "" {
		refBody, ok := helper.defsBackends[reference]
		if !ok {
			r := backendBody.MissingItemRange()
			return nil, newDiagErr(&r, "backend reference is not defined: "+reference)
		}

		if backendBody == nil {
			if attrName == "_init" { // initial definitions case
				backendBody = hclbody.MergeBodies(defaultBackend, refBody)
			} else { // plain reference without params
				return refBody, nil
			}
		} else { // with backend params - do not repeat referenced hcl stack, just the name
			if err = invalidRefinement(backendBody); err != nil {
				return nil, err
			}
			if err = invalidOriginRefinement(refBody, backendBody); err != nil {
				return nil, err
			}
			backendBody = hclbody.MergeBodies(
				hclbody.New(hclbody.NewContentWithAttrName("name", reference)),
				backendBody)
		}
	} else {
		labelBody := block.HCLBody()
		var labelSuffix string
		// anonymous backend based on a single attr, take the attr range instead
		if attrName != "" && reference == "" {
			labelBody, labelSuffix = refineAnonLabel(attrName, labelBody)
		}

		anonLabel := newAnonLabel(block.HCLBody()) + labelSuffix
		if backendBody == nil {
			backendBody = defaultBackend
		} else {
			backendBody = hclbody.MergeBodies(defaultBackend, backendBody)
		}

		backendBody, err = NewNamedBody(anonLabel, backendBody)
		if err != nil {
			return nil, err
		}
	}

	// configure backend with known endpoint url
	if attrValue != "" {
		backendBody = hclbody.MergeBodies(backendBody,
			hclbody.New(hclbody.NewContentWithAttrName("_backend_url", attrValue)))
	}

	// watch out for oauth blocks and nested backend definitions
	oauth2Backend, err := newOAuthBackend(helper, backendBody)
	if err != nil {
		return nil, err
	}

	if oauth2Backend != nil {
		wrapped := wrapOauth2Backend(oauth2Backend)
		backendBody = hclbody.MergeBodies(backendBody, wrapped)
	}

	return backendBody, nil
}

func getBackendBody(inline config.Inline) (string, hcl.Body, error) {
	var reference string

	content, _, diags := inline.HCLBody().PartialContent(inline.Schema(true))
	if diags.HasErrors() {
		return "", nil, diags
	}

	backends := content.Blocks.OfType(backend)
	if len(backends) > 0 {
		body := backends[0].Body
		if len(backends[0].Labels) > 0 {
			reference = backends[0].Labels[0]
		}
		return reference, body, nil
	}
	return reference, nil, nil
}

// TODO: circular dep check
func newOAuthBackend(helper *Helper, parent hcl.Body) (hcl.Body, error) {
	innerContent, err := contentByType(oauth2, parent)
	if err != nil {
		return nil, err
	}

	oauthBlocks := innerContent.Blocks.OfType(oauth2)
	if len(oauthBlocks) == 0 {
		return nil, nil
	}

	// oauth block exists, read out backend configuration
	oauthBody := oauthBlocks[0].Body
	conf := &config.OAuth2ReqAuth{}
	if diags := gohcl.DecodeBody(oauthBody, helper.context, conf); diags.HasErrors() {
		return nil, diags
	}

	return PrepareBackend(helper, "", conf.TokenEndpoint, conf)
}

func wrapOauth2Backend(content hcl.Body) hcl.Body {
	b := hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{
				Type: oauth2,
				Body: newBackendBlock(content),
			},
		},
	})
	return b
}

func newBackendBlock(content hcl.Body) hcl.Body {
	return hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{
				Type: backend,
				Body: content,
			},
		},
	})
}

func NewNamedBody(nameValue string, config hcl.Body) (hcl.Body, error) {
	if err := validLabel(nameValue, getRange(config)); err != nil {
		return nil, err
	}

	return hclbody.MergeBodies(
		config,
		hclbody.New(hclbody.NewContentWithAttrName("name", nameValue)),
	), nil
}

func newAnonLabel(body hcl.Body) string {
	const anon = "anonymous"
	itemRange := getRange(body)

	return fmt.Sprintf("%s_%d_%d", anon,
		itemRange.Start.Line,
		itemRange.Start.Column,
	)
}

func refineAnonLabel(attrName string, body hcl.Body) (labelBody hcl.Body, labelSuffix string) {
	labelBody = body
	if syntaxBody, ok := body.(*hclsyntax.Body); ok {
		if attr, exist := syntaxBody.Attributes[attrName]; exist {
			labelBody = hclbody.New(&hcl.BodyContent{MissingItemRange: attr.Expr.StartRange()})
		} else { // not defined, no line mapping possible
			labelSuffix += "_" + attrName
		}
	}
	return labelBody, labelSuffix
}

var invalidAttributes = []string{"disable_certificate_validation", "disable_connection_reuse", "http2", "max_connections", "openapi"}

func invalidRefinement(body hcl.Body) error {
	attrs, _ := body.JustAttributes()
	if attrs == nil {
		return nil
	}
	for _, name := range invalidAttributes {
		attr, exist := attrs[name]
		if exist {
			return newDiagErr(&attr.NameRange,
				fmt.Sprintf("backend reference: refinement for %q is not permitted", attr.Name))
		}
	}
	return nil
}

func invalidOriginRefinement(reference, params hcl.Body) error {
	const origin = "origin"
	refAttrs, _ := reference.JustAttributes()
	paramAttrs, _ := params.JustAttributes()

	refOrigin, _ := refAttrs[origin]
	paramOrigin, _ := paramAttrs[origin]

	if paramOrigin != nil && refOrigin != nil {
		if paramOrigin.Expr != refOrigin.Expr {
			return newDiagErr(&paramOrigin.Range, fmt.Sprintf("backend reference: origin must be equal"))
		}
	}
	return nil
}
