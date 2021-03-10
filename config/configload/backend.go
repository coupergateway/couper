package configload

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
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
