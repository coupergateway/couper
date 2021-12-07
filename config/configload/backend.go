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

// Backends is a sequence of Backend
type Backends []*Backend

type Backend struct {
	attr   *hclsyntax.Attribute
	Config hcl.Body
	name   string
}

func NewBackend(name string, config hcl.Body) *Backend {
	return &Backend{
		attr: &hclsyntax.Attribute{
			Name: "name",
			Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(name)},
		},
		name:   name,
		Config: config,
	}
}

var backendBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       backend,
			LabelNames: []string{"name"},
		},
	},
}

// WithName picks a matching backend configuration from the backend sequence and does
// some attribute enrichment to be able to filter with this attributes later on.
func (b Backends) WithName(name string) (hcl.Body, error) {
	if len(b) == 0 || name == "" {
		return nil, nil
	}

	for _, item := range b {
		if item.name == name {
			if syntaxBody, ok := item.Config.(*hclsyntax.Body); ok {
				if _, ok = syntaxBody.Attributes["name"]; ok {
					return item.Config, nil
				}
				// explicit set on every call since this could be affected by user content.
				// TODO: internal obj with hidden attributes
				syntaxBody.Attributes["name"] = item.attr
			}
			return item.Config, nil
		}
	}

	return nil, hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "backend reference is not defined: " + name,
	}}
}

// mergeBackendBodies appends the left side object with newly defined attributes or overrides already defined ones.
func mergeBackendBodies(definedBackends Backends, inline config.Inline) (hcl.Body, error) {
	var reference hcl.Body
	if beRef, ok := inline.(config.BackendReference); ok {
		r, err := getBackendReference(definedBackends, beRef)
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
		reference = MergeBodies([]hcl.Body{reference, hclbody.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange,
		})})
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
	refOverride, err := definedBackends.WithName(backendBlock.Labels[0])
	if err != nil {
		err.(hcl.Diagnostics)[0].Subject = &backendBlock.DefRange

		// Case: referenced backend is not defined in definitions.
		return nil, err
	}

	// link backend block name (label) to attribute 'name'
	if syntaxBody, ok := backendBlock.Body.(*hclsyntax.Body); ok {
		if refBody, ok := refOverride.(*hclsyntax.Body); ok {
			syntaxBody.Attributes[nameLabel] = refBody.Attributes[nameLabel]
		}
	}

	return MergeBodies([]hcl.Body{refOverride, backendBlock.Body}), nil
}

// getBackendReference tries to fetch a backend from `definitions`
// block by a reference name, e.g. `backend = "name"`.
func getBackendReference(definedBackends Backends, be config.BackendReference) (hcl.Body, error) {
	name := be.Reference()

	// backend string attribute just not set
	if name == "" {
		return nil, nil
	}

	reference, err := definedBackends.WithName(name)
	if err != nil {
		return nil, err // parse err
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

func newBackend(definedBackends Backends, inlineConfig config.Inline) (hcl.Body, error) {
	bend, err := mergeBackendBodies(definedBackends, inlineConfig)
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

	oauth2Backend, err := newOAuthBackend(definedBackends, bend)
	if err != nil {
		return nil, err
	}

	if oauth2Backend != nil {
		wrapped := hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{
			{Type: oauth2, Body: hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{
				{Type: backend, Body: oauth2Backend},
			}})},
		}})
		bend = MergeBodies([]hcl.Body{bend, wrapped})
	}

	return bend, nil
}

func newOAuthBackend(definedBackends Backends, parent hcl.Body) (hcl.Body, error) {
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

	oauthBackend, err := mergeBackendBodies(definedBackends, beConfig)
	if err != nil {
		return nil, err
	}

	return newBackend(definedBackends, &config.OAuth2ReqAuth{Remain: hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{Type: backend, Body: oauthBackend},
		},
	})})
}
