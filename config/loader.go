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

var typeMap = map[string]func(*logrus.Entry, hcl.Body) http.Handler{
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
		for _, backend := range server.Api.Backend {
			if isKeyword(backend.Name) {
				log.Fatalf("backend name not allowed, reserved keyword: '%s'", backend.Name)
			}
			if _, ok := backends[backend.Name]; ok {
				log.Fatalf("backend name must be unique: '%s'", backend.Name)
			}
			backends[backend.Name] = newBackend(backend.Kind, backend.Options, log)
		}

		server.Api.PathHandler = make(PathHandler)

		// map backends to endpoint
		apiSchema, _ := gohcl.ImpliedBodySchema(server.Api)
		endpoints := make(map[string]bool)
		for e, endpoint := range server.Api.Endpoint {
			config.Server[a].Api.Endpoint[e].Server = server // assign parent
			if endpoints[endpoint.Pattern] {
				log.Fatal("Duplicate endpoint: ", endpoint.Pattern)
			}

			endpoints[endpoint.Pattern] = true
			if endpoint.Backend != "" {
				if _, ok := backends[endpoint.Backend]; !ok {
					log.Fatalf("backend %q not found", endpoint.Backend)
				}
				server.Api.PathHandler[endpoint] = backends[endpoint.Backend]
				continue
			}
			// TODO: instead of passing the Server Scheme for backend block description, ask for definition via interface later on
			content, leftOver, diags := endpoint.Options.PartialContent(apiSchema)
			if diags.HasErrors() {
				for _, diag := range diags {
					if diag.Summary != "Missing name for backend" {
						log.Fatal(diags.Error())
					}
				}
			}
			endpoint.Options = leftOver

			if len(content.Blocks) == 0 {
				log.Fatal("expected backend attribute reference or block")
			}
			kind := content.Blocks[0].Labels[0]

			server.Api.PathHandler[endpoint] = newBackend(kind, content.Blocks[0].Body, log) // inline backend
		}

		// serve files
		if server.Files.DocumentRoot != "" {
			fileHandler, err := backend.NewFile(server.Files.DocumentRoot, log, server.Spa.BootstrapFile, server.Spa.Paths)
			if err != nil {
				log.Fatalf("Failed to load configuration: %s", err)
			}
			config.Server[a].FileHandler = fileHandler
		}
	}

	return &config
}

func newBackend(kind string, options hcl.Body, log *logrus.Entry) http.Handler {
	if !isKeyword(kind) {
		log.Fatalf("Invalid backend: %s", kind)
	}
	b := typeMap[strings.ToLower(kind)](log, options)

	return b
}

func isKeyword(other string) bool {
	_, yes := typeMap[other]
	return yes
}
