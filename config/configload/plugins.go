package configload

import (
	"fmt"
	goplugin "plugin"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/schema"
	"github.com/avenga/couper/plugins"
)

var pluginBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			LabelOptional: true,
			LabelNames:    []string{"name"},
			Type:          plugin,
		},
	},
}

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
			return err
		}

		sym, err := loadedPlugin.Lookup("Plugin")
		if err != nil {
			return err
		}

		if conf, impl := sym.(plugins.Config); impl {
			parentBlock, header, schemaBody := conf.Definition()
			if parentBlock != plugins.Definitions && parentBlock != plugins.Endpoint {
				return fmt.Errorf("extending the %s block type is not supported", parentBlock)
			}
			if parentBlock != "" && schemaBody != nil {
				if err = schema.Registry.Add(parentBlock, header, schemaBody); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
