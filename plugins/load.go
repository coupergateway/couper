package plugins

import (
	"context"
	"fmt"
	goplugin "plugin"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/schema"
)

var pluginBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			LabelOptional: true,
			LabelNames:    []string{"name"},
			Type:          "plugin",
		},
	},
}

type loaded struct {
	impl    interface{}
	schDefs []SchemaDefinition
}

var loadedPlugins = map[string]loaded{}

func Load(ctx *hcl.EvalContext, body hcl.Body) error {
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

		fileName := f.AsString()
		loadedPlugin, err := goplugin.Open(fileName)
		if err != nil {
			return err
		}

		sym, err := loadedPlugin.Lookup("Plugin")
		if err != nil {
			return err
		}

		if conf, impl := sym.(Config); impl {
			var schemaDefs []SchemaDefinition
			schemaCh := make(chan SchemaDefinition)
			readCtx, cancelRead := context.WithCancel(context.Background())
			go func() {
				for {
					select {
					case s := <-schemaCh:
						schemaDefs = append(schemaDefs, s)
					case <-readCtx.Done():
						return
					}

				}
			}()

			conf.Definition(schemaCh)
			cancelRead()

			for _, def := range schemaDefs {
				if def.Parent != Definitions && def.Parent != Endpoint {
					return fmt.Errorf("extending the %s block type is not supported", def.Parent)
				}
				if def.Parent != "" && def.Body != nil {
					schema.Registry.Add(def.BlockHeader, def.Body)
				}
			}

			loadedPlugins[fileName] = loaded{
				impl:    loadedPlugin,
				schDefs: schemaDefs,
			}
		}
	}

	return nil
}

func List() (result []string) {
	for k := range loadedPlugins {
		result = append(result, k)
	}
	return result
}
