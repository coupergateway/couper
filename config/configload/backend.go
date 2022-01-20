package configload

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/internal/seetie"
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

func NewBackendConfigBody(name string, config hcl.Body) (hcl.Body, error) {
	if err := validLabel(name, getRange(config)); err != nil {
		return nil, err
	}
	return hclbody.MergeBodies(
		defaultBackend,
		config,
		hclbody.New(newContentWithName(name)),
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

func newContentWithName(name string) *hcl.BodyContent {
	return &hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{
			"name": {
				Name: "name",
				Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(name)},
			},
		},
	}
}

// mergeBackendBodies appends the left side object with newly defined attributes or overrides already defined ones.
func mergeBackendBodies(loader *Loader, inline config.Inline) (hcl.Body, error) {
	var reference hcl.Body

	if beRef, ok := inline.(config.BackendReference); ok {
		if name := beRef.Reference(); name != "" {
			reference, ok = loader.defsBackends[name]
			if !ok {
				return nil, hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "backend reference is not defined: " + name,
				}}
			}
		}
	}

	content, _, diags := inline.HCLBody().PartialContent(inline.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	if content == nil {
		if reference != nil {
			return reference, nil
		}
		return nil, fmt.Errorf("configuration error: missing backend reference or inline definition")
	}

	// Apply current attributes to the referenced body.
	if len(content.Attributes) > 0 && reference != nil {
		reference = hclbody.MergeBodies(reference, hclbody.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange}),
		)
	}

	var backendBlock *hcl.Block
	if backends := content.Blocks.OfType(backend); len(backends) > 0 {
		backendBlock = backends[0]
	} else {
		return reference, nil
	}

	// Case: `backend {}`, anonymous backend.
	if len(backendBlock.Labels) == 0 /*|| backendBlock.Labels[0] == ""*/ {
		if backendBlock.Body == nil {
			return nil, nil
		}

		name := newAnonLabel(backendBlock.Body)

		body, _ := NewBackendConfigBody(name, backendBlock.Body)
		loader.anonBackends[name] = body

		return body, nil
	}

	// Case: `backend "reference" {}`, referenced backend.
	refOverride, ok := loader.defsBackends[backendBlock.Labels[0]]
	if !ok {
		// Case: referenced backend is not defined in definitions.
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "backend reference is not defined: " + backendBlock.Labels[0],
			Subject:  &backendBlock.DefRange,
		}}
	}

	// link backend block name (label) to attribute 'name'
	if syntaxBody, ok := backendBlock.Body.(*hclsyntax.Body); ok {
		if refBody, ok := refOverride.(*hclsyntax.Body); ok {
			syntaxBody.Attributes[nameLabel] = refBody.Attributes[nameLabel]
		}
	}

	return hclbody.MergeBodies(refOverride, backendBlock.Body), nil
}

func newBackend(loader *Loader, inline config.Inline) (hcl.Body, error) {
	bend, err := mergeBackendBodies(loader, inline)
	if err != nil {
		return nil, err
	}

	if bend == nil {
		return loader.anonBackends[anonDefName], nil
	}

	oauth2Backend, err := newOAuthBackendConfigBody(loader, bend)
	if err != nil {
		return nil, err
	}

	if oauth2Backend != nil {
		wrapped := wrapOauth2Backend(oauth2Backend)
		bend = hclbody.MergeBodies(bend, wrapped)
	}

	return bend, nil
}

func newOAuthBackendConfigBody(loader *Loader, parent hcl.Body) (hcl.Body, error) {
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

	backendContent, err := contentByType(backend, oauthBody)
	if err != nil {
		return nil, err
	}

	oauthConfig := &config.Backend{Remain: hclbody.New(backendContent)}

	// reference ?
	attrs, _ := oauthBody.JustAttributes()
	if attrs != nil && attrs[backend] != nil {
		val, _ := attrs[backend].Expr.Value(loader.context)

		if ref := seetie.ValueToString(val); ref != "" {
			oauthConfig.Name = ref
		}
	}

	// without ref, create anon label
	if oauthConfig.Name == "" {
		oauthConfig.Name = newAnonLabel(oauthBody)
	}

	oauthBackend, err := mergeBackendBodies(loader, oauthConfig)
	if err != nil {
		return nil, err
	}

	// possible recursive call, required for nested oauth backend blocks
	return newBackend(loader, &config.OAuth2ReqAuth{
		Remain: hclbody.New(&hcl.BodyContent{
			Blocks: []*hcl.Block{
				{
					Type:   backend,
					Body:   oauthBackend,
					Labels: []string{oauthConfig.Name},
				},
			},
		}),
	})
}

func wrapOauth2Backend(content hcl.Body) hcl.Body {
	b := hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{
				Type: oauth2,
				Body: hclbody.New(&hcl.BodyContent{
					Blocks: []*hcl.Block{
						{
							Type: backend,
							Body: content,
						},
					},
				}),
			},
		},
	})
	return b
}
