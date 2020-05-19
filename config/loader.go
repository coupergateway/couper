package config

import (
	"log"
	"net/http"
	"reflect"

	"github.com/hashicorp/hcl/v2/hclsimple"

	"go.avenga.cloud/couper/gateway/backend"
)

var typeMap = map[string]reflect.Type{
	"Proxy": reflect.TypeOf(backend.Proxy{}),
}

func Load(name string) *Gateway {
	var config Gateway
	err := hclsimple.DecodeFile(name, nil, &config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	for i, frontend := range config.Frontends { // TODO: additional conf etc
		backend := reflect.New(typeMap[frontend.Endpoint.Type]).Interface()
		if handler, ok := backend.(http.Handler); ok {
			config.Frontends[i].Endpoint.Backend = handler
		}
	}

	return &config
}
