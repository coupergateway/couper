package config

import (
	"log"
	"net/http"
	"reflect"

	"github.com/hashicorp/hcl/v2/gohcl"
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

	for f, frontend := range config.Frontends {
		for e, endpoint := range frontend.Endpoint {
			val := reflect.New(typeMap[endpoint.Backend.Type])
			diags := gohcl.DecodeBody(endpoint.Backend.Options, nil, val.Interface())
			if diags.HasErrors() {
				log.Fatal(diags.Error())
			}
			if handler, ok := val.Interface().(interface{ Init() }); ok {
				handler.Init() // TODO: NewWithHCLBodyFuncWrap
			}
			if handler, ok := val.Interface().(http.Handler); ok {
				config.Frontends[f].Endpoint[e].Backend.instance = handler
				config.Frontends[f].Endpoint[e].Frontend = frontend // assign parent
			}
		}
	}

	return &config
}
