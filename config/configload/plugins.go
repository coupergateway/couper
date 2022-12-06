package configload

import (
	goplugin "plugin"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/plugins"
)

type blockSchema map[*hcl.BlockHeaderSchema]*hcl.BodySchema

var pluginBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			LabelOptional: true,
			LabelNames:    []string{"name"},
			Type:          plugin,
		},
	},
}

var pluginSchemaExtensions = make(map[string][]blockSchema)

func LoadPlugins(ctx *hcl.EvalContext, body hcl.Body) error {
	pluginContent, _, diagnostics := body.PartialContent(pluginBlockSchema)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	if len(pluginContent.Blocks) == 0 {
		return nil
	}

	for _, block := range pluginContent.Blocks {
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return diags
		}
		f, diags := attrs["file"].Expr.Value(ctx)
		if diags.HasErrors() {
			return diags
		}

		loadedPlugin, err := goplugin.Open(f.AsString())
		if err != nil {
			println(err.Error())
			return err
		}

		sym, err := loadedPlugin.Lookup("CouperPlugin")
		if err != nil {
			return err
		}

		if schemaRegisterer, impl := sym.(plugins.Config); impl {
			parentBlock, header, schema := schemaRegisterer.Register()
			if parentBlock != "" && schema != nil {

				pluginSchemaExtensions[parentBlock] = append(pluginSchemaExtensions[parentBlock], blockSchema{
					header: schema,
				})
			}
		}
	}

	return nil
}
