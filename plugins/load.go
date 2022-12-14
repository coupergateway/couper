package plugins

import (
	"context"
	"fmt"
	goplugin "plugin"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

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

type Loaded struct {
	impl   interface{}
	Schema []SchemaDefinition
}

var loadedPlugins = map[string]Loaded{}

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

			loadedPlugins[fileName] = Loaded{
				impl:   sym,
				Schema: schemaDefs[:],
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

func Get(mp MountPoint) []*Loaded {
	var filtered []Loaded
	for _, v := range loadedPlugins {
		for _, s := range v.Schema {
			if s.Parent == mp {
				filtered = append(filtered, v)
				break
			}
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// DecodeForType Iface
	var l []*Loaded
	for _, f := range filtered {
		var filteredSchema []SchemaDefinition
		for _, sd := range f.Schema {
			if sd.Parent == mp {
				filteredSchema = append(filteredSchema, sd)
			}
		}
		l = append(l, &Loaded{
			impl:   f.impl,
			Schema: filteredSchema,
		})
	}
	return l
}

func (l *Loaded) DecodeBody(ctx *hcl.EvalContext, body hcl.Body) error {
	decodeFn := func(ref any) error {
		val := reflect.ValueOf(ref)
		if val.Kind() != reflect.Pointer || val.IsNil() {
			return fmt.Errorf("invalid type: %s", reflect.TypeOf(ref))
		}
		diags := gohcl.DecodeBody(body, ctx, ref)
		if diags.HasErrors() {
			return diags
		}
		return nil
	}

	return l.impl.(Config).Decode(decodeFn)
}
