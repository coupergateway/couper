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

	if strings.HasSuffix(attrName, "_backend") && attrValue != "" { // specific attribute overrides; prefer
		reference = attrValue
	}

	if reference != "" {
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
			backendBody = newBodyWithName(reference, backendBody)
			// no child blocks are allowed, so no need to try to wrap with oauth2 or token request
			return backendBody, nil
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

		backendBody = newBodyWithName(anonLabel, backendBody)
	}

	// watch out for oauth blocks and nested backend definitions
	backendBody, err = wrapOAuthBackend(helper, backendBody)
	if err != nil {
		return nil, err
	}

	// watch out for beta_token_request blocks and nested backend definitions
	return wrapTokenRequestBackend(helper, backendBody)
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

// wrapOAuthBackend prepares a nested backend within a backend-oauth2 block.
// TODO: Check a possible circular dependency with given parent backend(s).
func wrapOAuthBackend(helper *helper, parent hcl.Body) (hcl.Body, error) {
	innerContent, _, diags := parent.PartialContent(config.OAuthBlockSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	oauthBlocks := innerContent.Blocks.OfType(oauth2)
	if len(oauthBlocks) == 0 {
		return parent, nil
	}

	// oauth block exists, read out backend configuration
	oauthBody := oauthBlocks[0].Body
	conf := &config.OAuth2ReqAuth{}
	if diags = gohcl.DecodeBody(oauthBody, helper.context, conf); diags.HasErrors() {
		return nil, diags
	}

	backendBody, err := PrepareBackend(helper, "", conf.TokenEndpoint, conf)
	if err != nil {
		return nil, err
	}

	wrapped := wrapBlock(oauth2, oauthBlocks[0].Labels, backendBody)
	parent = hclbody.MergeBodies(parent, wrapped)

	return parent, nil
}

func checkTokenRequestLabels(backendBody hcl.Body, unique map[string]struct{}) error {
	ic, _, diags := backendBody.PartialContent(config.TokenRequestBlockSchema)
	if diags.HasErrors() {
		return diags
	}

	trbs := ic.Blocks.OfType(tokenRequest)
	for _, trb := range trbs {
		label := defaultNameLabel
		r := &trb.DefRange
		if len(trb.Labels) > 0 {
			label = trb.Labels[0]
			r = &trb.LabelRanges[0]
			if err := validLabel(label, r); err != nil {
				return err
			}
		}

		if err := uniqueLabelName("token request", unique, label, r); err != nil {
			return err
		}

	}
	return nil
}

// wrapTokenRequestBackend prepares a nested backend within each backend-tokenRequest block.
// TODO: Check a possible circular dependency with given parent backend(s).
func wrapTokenRequestBackend(helper *helper, parent hcl.Body) (hcl.Body, error) {
	unique := map[string]struct{}{}
	if mb, ok := parent.(hclbody.MergedBodies); ok {
		for _, bo := range mb {
			err := checkTokenRequestLabels(bo, unique)
			if err != nil {
				return nil, err
			}
		}
	} else {
		err := checkTokenRequestLabels(parent, unique)
		if err != nil {
			return nil, err
		}
	}

	innerContent, _, diags := parent.PartialContent(config.TokenRequestBlockSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	tokenRequestBlocks := innerContent.Blocks.OfType(tokenRequest)
	if len(tokenRequestBlocks) == 0 {
		return parent, nil
	}

	// beta_token_request block exists, read out backend configuration
	for _, tokenRequestBlock := range tokenRequestBlocks {
		tokenRequestBody := tokenRequestBlock.Body
		conf := &config.TokenRequest{}
		if diags = gohcl.DecodeBody(tokenRequestBody, helper.context, conf); diags.HasErrors() {
			return nil, diags
		}

		content, leftOvers, diags := conf.Remain.PartialContent(conf.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}

		if err := verifyBodyAttributes(tokenRequest, content); err != nil {
			return nil, err
		}

		hclbody.RenameAttribute(content, "headers", "set_request_headers")
		hclbody.RenameAttribute(content, "query_params", "set_query_params")
		conf.Remain = hclbody.MergeBodies(leftOvers, hclbody.New(content))

		tokenRequestBackend, berr := PrepareBackend(helper, "", conf.URL, conf)
		if berr != nil {
			return nil, berr
		}

		wrapped := wrapBlock(tokenRequest, tokenRequestBlock.Labels,
			hclbody.MergeBodies(conf.Remain, tokenRequestBackend))
		parent = hclbody.MergeBodies(parent, wrapped)
	}

	return parent, nil
}

func wrapBlock(blockType string, labels []string, content hcl.Body) hcl.Body {
	return hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{
				Body:   newBlock(backend, content),
				Labels: labels,
				Type:   blockType,
			},
		},
	})
}

func newBlock(blockType string, content hcl.Body) hcl.Body {
	return hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{
				Body: content,
				Type: blockType,
			},
		},
	})
}

func newBodyWithName(nameValue string, config hcl.Body) hcl.Body {
	return hclbody.MergeBodies(
		config,
		hclbody.NewHCLSyntaxBodyWithStringAttr("name", nameValue),
	)
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
