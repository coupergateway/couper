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

func newDefaultBackend() *hclsyntax.Body {
	return &hclsyntax.Body{
		Attributes: map[string]*hclsyntax.Attribute{
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
	}
}

// PrepareBackend is a method which is mandatory to call for preparing any kind of backend.
// This applies to defined, reference, anonymous and endpoint/url related configurations.
// This method will be called recursively and is used as wrapped injector for
// access-control backends via config.PrepareBackendFunc.
func PrepareBackend(helper *helper, attrName, attrValue string, block config.Body) (*hclsyntax.Body, error) {
	var reference string // backend definitions
	var backendBody *hclsyntax.Body
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
			r := block.HCLBody().SrcRange
			if backendBody != nil {
				r = backendBody.SrcRange
			}
			return nil, newDiagErr(&r, fmt.Sprintf("referenced backend %q is not defined", reference))
		}

		if backendBody == nil {
			if attrName == "_init" { // initial definitions case
				backendBody = hclbody.MergeBodies(refBody, newDefaultBackend(), false)
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
			setName(reference, backendBody)
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
			backendBody = newDefaultBackend()
		} else {
			switch labelSuffix {
			case "_jwks_uri_backend", "_token_backend", "_userinfo_backend":
				copied := *backendBody
				// create new Attributes to allow different name later
				copied.Attributes = make(map[string]*hclsyntax.Attribute, len(backendBody.Attributes))
				for k, v := range backendBody.Attributes {
					copied.Attributes[k] = v
				}
				backendBody = hclbody.MergeBodies(&copied, newDefaultBackend(), false)
			default:
				// with OIDC this is used for _configuration_backend
				backendBody = hclbody.MergeBodies(backendBody, newDefaultBackend(), false)
			}
		}

		setName(anonLabel, backendBody)
	}

	// watch out for oauth blocks and nested backend definitions
	backendBody, err = setOAuth2Backend(helper, backendBody)
	if err != nil {
		return nil, err
	}

	// watch out for beta_token_request blocks and nested backend definitions
	return setTokenRequestBackend(helper, backendBody)
}

// getBackendReference reads a referenced backend name and the refined backend block content if any.
func getBackendReference(b config.Body) (string, *hclsyntax.Body, error) {
	var reference string

	backends := hclbody.BlocksOfType(b.HCLBody(), backend)
	if len(backends) == 0 {
		if beref, ok := b.(config.BackendReference); ok {
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

// setOAuth2Backend prepares a nested backend within a backend-oauth2 block.
// TODO: Check a possible circular dependency with given parent backend(s).
func setOAuth2Backend(helper *helper, parent *hclsyntax.Body) (*hclsyntax.Body, error) {
	oauthBlocks := hclbody.BlocksOfType(parent, oauth2)
	if len(oauthBlocks) == 0 {
		return parent, nil
	}

	// oauth block exists, read out backend configuration
	oauthBody := oauthBlocks[0].Body
	conf := &config.OAuth2ReqAuth{}
	if diags := gohcl.DecodeBody(oauthBody, helper.context, conf); diags.HasErrors() {
		return nil, diags
	}

	backendBody, err := PrepareBackend(helper, "", conf.TokenEndpoint, conf)
	if err != nil {
		return nil, err
	}

	if len(hclbody.BlocksOfType(oauthBody, backend)) == 0 {
		// only add backend block, if not already there
		backendBlock := &hclsyntax.Block{
			Type: backend,
			Body: backendBody,
		}
		oauthBody.Blocks = append(oauthBody.Blocks, backendBlock)
	}

	return parent, nil
}

func checkTokenRequestLabels(trbs []*hclsyntax.Block, unique map[string]struct{}) error {
	for _, trb := range trbs {
		label := config.DefaultNameLabel
		dr := trb.DefRange()
		r := &dr
		if len(trb.Labels) > 0 {
			label = trb.Labels[0]
			r = &trb.LabelRanges[0]
			if err := validLabel(label, r); err != nil {
				return err
			}
		} else {
			// add "default" label if no label is configured
			trb.Labels = append(trb.Labels, label)
		}

		if err := uniqueLabelName("token request", unique, label, r); err != nil {
			return err
		}

	}
	return nil
}

// setTokenRequestBackend prepares a nested backend within each backend-tokenRequest block.
// TODO: Check a possible circular dependency with given parent backend(s).
func setTokenRequestBackend(helper *helper, parent *hclsyntax.Body) (*hclsyntax.Body, error) {
	tokenRequestBlocks := hclbody.BlocksOfType(parent, tokenRequest)
	if len(tokenRequestBlocks) == 0 {
		return parent, nil
	}

	unique := map[string]struct{}{}
	err := checkTokenRequestLabels(tokenRequestBlocks, unique)
	if err != nil {
		return nil, err
	}

	// beta_token_request block exists, read out backend configuration
	for _, tokenRequestBlock := range tokenRequestBlocks {
		tokenRequestBody := tokenRequestBlock.Body
		conf := &config.BetaTokenRequest{}
		if diags := gohcl.DecodeBody(tokenRequestBody, helper.context, conf); diags.HasErrors() {
			return nil, diags
		}

		if err = verifyBodyAttributes(tokenRequest, tokenRequestBody); err != nil {
			return nil, err
		}

		hclbody.RenameAttribute(tokenRequestBody, "headers", "set_request_headers")
		hclbody.RenameAttribute(tokenRequestBody, "query_params", "set_query_params")

		backendBody, berr := PrepareBackend(helper, "", conf.URL, conf)
		if berr != nil {
			return nil, berr
		}

		if len(hclbody.BlocksOfType(tokenRequestBody, backend)) == 0 {
			// only add backend block, if not already there
			backendBlock := &hclsyntax.Block{
				Type: backend,
				Body: backendBody,
			}
			tokenRequestBody.Blocks = append(tokenRequestBody.Blocks, backendBlock)
		}
	}

	return parent, nil
}

func setName(nameValue string, backendBody *hclsyntax.Body) {
	backendBody.Attributes["name"] = &hclsyntax.Attribute{
		Name: "name",
		Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(nameValue)},
	}
}

func newAnonLabel(body *hclsyntax.Body, labelRange *hcl.Range) string {
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

func refineAnonLabel(attrName string, body *hclsyntax.Body) (labelRange *hcl.Range, labelSuffix string) {
	if attr, exist := body.Attributes[attrName]; exist {
		labelRange = &attr.NameRange
	}
	labelSuffix += "_" + attrName
	return labelRange, labelSuffix
}
