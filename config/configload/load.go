package configload

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/eval"
)

const (
	backend     = "backend"
	definitions = "definitions"
	server      = "server"
	settings    = "settings"
)

func LoadFile(filePath string) (*config.CouperFile, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	src, err := ioutil.ReadFile(path.Join(wd, filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return LoadBytes(src, filePath)
}

func LoadBytes(src []byte, filePath string) (*config.CouperFile, error) {
	filename := filepath.Base(filePath)
	body, diags := parser.Load(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	return LoadConfig(body, src)
}

func LoadConfig(body hcl.Body, src []byte) (*config.CouperFile, error) {
	defaults := config.DefaultSettings
	file := &config.CouperFile{
		Bytes:       src,
		Context:     eval.NewENVContext(src),
		Definitions: &config.Definitions{},
		Settings:    &defaults,
	}

	fileSchema, _ := gohcl.ImpliedBodySchema(file)
	content, diags := body.Content(fileSchema)
	if content == nil {
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	// reading possible reference definitions first. Those are the base for refinement merges during server block read out.
	for _, outerBlock := range content.Blocks {
		switch outerBlock.Type {
		case definitions:
			if diags = gohcl.DecodeBody(outerBlock.Body, file.Context, file.Definitions); diags.HasErrors() {
				return nil, diags
			}
		case settings:
			if diags = gohcl.DecodeBody(outerBlock.Body, file.Context, file.Settings); diags.HasErrors() {
				return nil, diags
			}
		}
	}

	// reading per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range content.Blocks.OfType(server) {
		srv := &config.Server{}
		if diags = gohcl.DecodeBody(serverBlock.Body, file.Context, srv); diags.HasErrors() {
			return nil, diags
		}

		// api block(s)
		for _, apiBlock := range []*config.Api{srv.API} {
			if apiBlock == nil {
				continue
			}
			merged, err := mergeBackendBodies(file.Definitions, apiBlock)
			if err != nil {
				return nil, err
			}

			// empty bodies would be removed with a hcl.Merge.. later on.
			bodies := []hcl.Body{merged}
			if err = refineEndpoints(file.Definitions, bodies, apiBlock.Endpoints); err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		// TODO: free endpoints
		//if err := refineEndpoints(file.Definitions, nil, srv.Endpoints); err != nil {
		//	return nil, err
		//}

		file.Server = append(file.Server, srv)
	}

	if len(file.Server) == 0 {
		return nil, fmt.Errorf("missing server definition")
	}

	return file, nil
}

func mergeBackendBodies(definitions *config.Definitions, inlineBackend config.Inline) (hcl.Body, error) {
	reference, err := definitions.BackendWithName(inlineBackend.Reference())
	if err != nil {
		return nil, err
	}

	content, _, diags := inlineBackend.Body().PartialContent(inlineBackend.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	var bodies []hcl.Body
	var backends hcl.Blocks
	if content != nil {
		backends = content.Blocks.OfType(backend)
	}

	if reference != nil {
		if content != nil && len(backends) > 0 {
			return nil, fmt.Errorf("configuration error: inlineBackend reference and inline definition")
		}
		bodies = append(bodies, reference.Remain)
	}

	if len(backends) > 0 {
		if len(backends[0].Labels) > 0 {
			reference, err = definitions.BackendWithName(backends[0].Labels[0])
			if err != nil {
				return nil, err
			}
			bodies = append(bodies, reference.Remain)
		}
		// TODO: ep path

		bodies = append(bodies, backends[0].Body)
	}

	return MergeBodies(bodies), nil
}

func refineEndpoints(definitions *config.Definitions, parents []hcl.Body, endpoints config.Endpoints) error {
	for _, endpoint := range endpoints {
		merged, err := mergeBackendBodies(definitions, endpoint)
		if err != nil {
			return err
		}
		endpoint.Remain = MergeBodies(append(parents, merged))
	}
	return nil
}
