package configload

import (
	"fmt"
	"strings"

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

// PrepareBackend is a method which is mandatory to call for preparing any kind of backend.
// This applies to defined, reference, anonymous and endpoint/url related configurations.
// This method will be called recursively and is used as wrapped injector for
// access-control backends via config.PrepareBackendFunc.
func PrepareBackend(helper *helper, attrName, attrValue string, block config.Inline) (hcl.Body, error) {
	var reference string // backend definitions
	var backendBody hcl.Body
	var err error

	reference, backendBody, err = getBackendReference(block)
	if err != nil {
		return nil, err
	}

	if reference != "" {
		if strings.HasSuffix(attrName, "_backend") && attrValue != "" { // specific attribute overrides; prefer
			reference = attrValue
		}

		refBody, ok := helper.defsBackends[reference]
		if !ok {
			r := block.HCLBody().MissingItemRange()
			if backendBody != nil {
				r = backendBody.MissingItemRange()
			}
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
	} else { // anonymous backend block
		var labelRange *hcl.Range
		var labelSuffix string
		// anonymous backend based on a single attr, take the attr range instead
		if attrName != "" && reference == "" {
			labelRange, labelSuffix = refineAnonLabel(attrName, block.HCLBody())
		}

		anonLabel := newAnonLabel(block.HCLBody(), labelRange) + labelSuffix
		// ensure our default settings
		if backendBody == nil {
			backendBody = defaultBackend
		} else {
			backendBody = hclbody.MergeBodies(defaultBackend, backendBody)
		}

		backendBody, err = newBodyWithName(anonLabel, backendBody)
		if err != nil {
			return nil, err
		}
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

	// watch out for token_request blocks and nested backend definitions
	tokenRequestBackend, err := newTokenRequestBackend(helper, backendBody)
	if err != nil {
		return nil, err
	}

	if tokenRequestBackend != nil {
		wrapped := wrapTokenRequestBackend(tokenRequestBackend)
		backendBody = hclbody.MergeBodies(backendBody, wrapped)
	}

	return backendBody, nil
}

// getBackendReference reads a referenced backend name and the refined backend block content if any.
func getBackendReference(inline config.Inline) (string, hcl.Body, error) {
	var reference string

	content, _, diags := inline.HCLBody().PartialContent(inline.Schema(true))
	if diags.HasErrors() {
		return "", nil, diags
	}

	backends := content.Blocks.OfType(backend)
	if len(backends) == 0 {
		if beref, ok := inline.(config.BackendReference); ok {
			if beref.Reference() != "" {
				reference = beref.Reference()
			}
		}
		return reference, nil, nil
	}

	body := backends[0].Body
	if len(backends[0].Labels) > 0 {
		reference = backends[0].Labels[0]
	}
	return reference, body, nil
}

// newOAuthBackend prepares a nested backend within a backend-oauth2 block.
// TODO: Check a possible circular dependency with given parent backend(s).
func newOAuthBackend(helper *helper, parent hcl.Body) (hcl.Body, error) {
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

// newTokenRequestBackend prepares a nested backend within a backend-tokenRequest block.
// TODO: Check a possible circular dependency with given parent backend(s).
func newTokenRequestBackend(helper *helper, parent hcl.Body) (hcl.Body, error) {
	innerContent, err := contentByType(tokenRequest, parent)
	if err != nil {
		return nil, err
	}

	tokenRequestBlocks := innerContent.Blocks.OfType(tokenRequest)
	if len(tokenRequestBlocks) == 0 {
		return nil, nil
	}

	// token_request block exists, read out backend configuration
	tokenRequestBody := tokenRequestBlocks[0].Body
	conf := &config.TokenRequest{}
	if diags := gohcl.DecodeBody(tokenRequestBody, helper.context, conf); diags.HasErrors() {
		return nil, diags
	}

	return PrepareBackend(helper, "", conf.URL, conf)
}

func wrapTokenRequestBackend(content hcl.Body) hcl.Body {
	b := hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{
				Type: tokenRequest,
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

func newBodyWithName(nameValue string, config hcl.Body) (hcl.Body, error) {
	if err := validLabel(nameValue, getRange(config)); err != nil {
		return nil, err
	}

	return hclbody.MergeBodies(
		config,
		hclbody.New(hclbody.NewContentWithAttrName("name", nameValue)),
	), nil
}

func newAnonLabel(body hcl.Body, labelRange *hcl.Range) string {
	const anon = "anonymous"
	itemRange := labelRange
	if itemRange == nil {
		itemRange = getRange(body)
	}

	return fmt.Sprintf("%s_%d_%d", anon,
		itemRange.Start.Line,
		itemRange.Start.Column,
	)
}

func refineAnonLabel(attrName string, body hcl.Body) (labelRange *hcl.Range, labelSuffix string) {
	if syntaxBody, ok := body.(*hclsyntax.Body); ok {
		if attr, exist := syntaxBody.Attributes[attrName]; exist {
			labelRange = &attr.NameRange
		}
		labelSuffix += "_" + attrName
	}
	return labelRange, labelSuffix
}
