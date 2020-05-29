package config

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"

	"go.avenga.cloud/couper/gateway/backend"
)

var typeMap = map[string]func(*logrus.Entry) http.Handler{
	// "Proxy": reflect.TypeOf(backend.Proxy{}),
	"proxy": backend.NewProxy(),
}

func Load(name string, log *logrus.Entry) *Gateway {
	var config Gateway
	err := hclsimple.DecodeFile(name, nil, &config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	for f, frontend := range config.Frontends {
		for e, endpoint := range frontend.Endpoint {
			handler := typeMap[strings.ToLower(endpoint.Backend.Type)](log)
			diags := gohcl.DecodeBody(endpoint.Backend.Options, nil, handler)
			if diags.HasErrors() {
				log.Fatal(diags.Error())
			}
			config.Frontends[f].Endpoint[e].Backend.instance = handler
			config.Frontends[f].Endpoint[e].Frontend = frontend // assign parent
		}
	}

	return &config
}
