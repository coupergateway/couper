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
	subject := config.MissingItemRange()
	if diags := validLabel(name, &subject); diags != nil {
		return nil, diags
	}

	content := &hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{
			"name": {
				Name: "name",
				Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(name)},
			},
		},
	}

	return hclbody.MergeBodies(defaultBackend, config, hclbody.New(content)), nil
}

// mergeBackendBodies appends the left side object with newly defined attributes or overrides already defined ones.
func mergeBackendBodies(loader *Loader, inline config.Inline) (hcl.Body, error) {
	var reference hcl.Body
	if beRef, ok := inline.(config.BackendReference); ok {
		r, err := getBackendReference(loader, beRef)
		if err != nil {
			return nil, err
		}
		reference = r
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
	if len(backendBlock.Labels) == 0 {
		return backendBlock.Body, nil
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

// getBackendReference tries to fetch a backend from `definitions`
// block by a reference name, e.g. `backend = "name"`.
func getBackendReference(loader *Loader, be config.BackendReference) (hcl.Body, error) {
	name := be.Reference()

	// backend string attribute just not set
	if name == "" {
		return nil, nil
	}

	reference, ok := loader.defsBackends[name]
	if !ok {
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "backend reference is not defined: " + name,
		}}
	}

	// a name is given but we have no definition
	if body, ok := be.(config.Inline); ok {
		if b := body.HCLBody(); reference == nil && b != nil {
			r := b.MissingItemRange()
			return nil, hcl.Diagnostics{&hcl.Diagnostic{
				Subject: &r,
				Summary: fmt.Sprintf("backend reference '%s' is not defined", name),
			}}
		}
	}

	return reference, nil
}

func newBackend(loader *Loader, inlineConfig config.Inline) (hcl.Body, error) {
	bend, err := mergeBackendBodies(loader, inlineConfig)
	if err != nil {
		return nil, err
	}

	if bend == nil {
		// Create a default backend
		bend = hclbody.New(&hcl.BodyContent{
			Attributes: map[string]*hcl.Attribute{
				"name": {
					Name: "name",
					Expr: &hclsyntax.LiteralValueExpr{
						Val: cty.StringVal(defaultNameLabel),
					},
				},
			},
		})
	}

	oauth2Backend, err := newOAuthBackend(loader, bend)
	if err != nil {
		return nil, err
	}

	if oauth2Backend != nil {
		wrapped := hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{
			{Type: oauth2, Body: hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{
				{Type: backend, Body: oauth2Backend},
			}})},
		}})
		bend = hclbody.MergeBodies(bend, wrapped)
	}

	return bend, nil
}

func newOAuthBackend(loader *Loader, parent hcl.Body) (hcl.Body, error) {
	innerContent, err := contentByType(oauth2, parent)
	if err != nil {
		return nil, err
	}

	oauthBlocks := innerContent.Blocks.OfType(oauth2)
	if len(oauthBlocks) == 0 {
		return nil, nil
	}

	backendContent, err := contentByType(backend, oauthBlocks[0].Body)
	if err != nil {
		return nil, err
	}

	beConfig := &config.Backend{Remain: hclbody.New(backendContent)}

	attrs, _ := oauthBlocks[0].Body.JustAttributes()
	if attrs != nil && attrs["backend"] != nil {
		val, _ := attrs["backend"].Expr.Value(nil)

		if ref := seetie.ValueToString(val); ref != "" {
			beConfig.Name = ref
		}
	}

	oauthBackend, err := mergeBackendBodies(loader, beConfig)
	if err != nil {
		return nil, err
	}

	return newBackend(loader, &config.OAuth2ReqAuth{Remain: hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{Type: backend, Body: oauthBackend},
		},
	})})
}
