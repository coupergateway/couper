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

	for a, application := range config.Applications {
		// create backends
		for _, backend := range application.Backend {
			if isKeyword(backend.Name) {
				log.Fatalf("backend name not allowed, reserved keyword: '%s'", backend.Name)
			}
			if _, ok := backends[backend.Name]; ok {
				log.Fatalf("backend name must be unique: '%s'", backend.Name)
			}
			backends[backend.Name] = newBackend(backend.Kind, backend.Options, log)
		}

		// map backends to path
		for p, path := range application.Path {
			var handler http.Handler
			if h, ok := backends[path.Kind]; ok {
				handler = h
			} else {
				handler = newBackend(path.Kind, path.Options, log) // inline backend
			}
			config.Applications[a].Path[p].Backend = handler
			config.Applications[a].Path[p].Application = application // assign parent
		}

		// serve files
		if application.Files.DocumentRoot != "" {
			fileHandler := backend.NewFile(application.Files.DocumentRoot, log)
			config.Applications[a].instance = fileHandler
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
