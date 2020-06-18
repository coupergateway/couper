package config

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"

	"go.avenga.cloud/couper/gateway/backend"
)

var typeMap = map[string]func(*logrus.Entry) http.Handler{
	"proxy": backend.NewProxy(),
}

func Load(name string, log *logrus.Entry) *Gateway {
	var config Gateway
	err := hclsimple.DecodeFile(name, nil, &config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	backends := make(map[string]http.Handler)

	for a, server := range config.Server {
		// create backends
		for _, backend := range server.Backend {
			if isKeyword(backend.Name) {
				log.Fatalf("backend name not allowed, reserved keyword: '%s'", backend.Name)
			}
			if _, ok := backends[backend.Name]; ok {
				log.Fatalf("backend name must be unique: '%s'", backend.Name)
			}
			backends[backend.Name] = newBackend(backend.Kind, backend.Options, log)
		}

		server.PathHandler = make(PathHandler)

		// map backends to path
		serverSchema, _ := gohcl.ImpliedBodySchema(server)
		for p, path := range server.Path {
			config.Server[a].Path[p].Server = server // assign parent

			if path.Backend != "" {
				if _, ok := backends[path.Backend]; !ok {
					log.Fatalf("backend %q not found", path.Backend)
				}
				server.PathHandler[path] = backends[path.Backend]
				continue
			}
			// TODO: instead of passing the Server Scheme for backend block description, ask for definition via interface later on
			content, leftOver, _ := path.Options.PartialContent(serverSchema)
			path.Options = leftOver

			if len(content.Blocks) == 0 {
				log.Fatal("expected backend attribute reference or block")
			}
			// TODO: check len && type :)
			kind := content.Blocks[0].Labels[0]
			println(kind)

			server.PathHandler[path] = newBackend(kind, content.Blocks[0].Body, log) // inline backend
		}

		// serve files
		if server.Files.DocumentRoot != "" {
			fileHandler := backend.NewFile(server.Files.DocumentRoot, log)
			config.Server[a].instance = fileHandler
		}

	}

	return &config
}

func newBackend(kind string, options hcl.Body, log *logrus.Entry) http.Handler {
	b := typeMap[strings.ToLower(kind)](log)
	diags := gohcl.DecodeBody(options, nil, b)
	if diags.HasErrors() {
		log.Fatal(diags.Error())
	}
	return b
}

func isKeyword(other string) bool {
	_, yes := typeMap[other]
	return yes
}
