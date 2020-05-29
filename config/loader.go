package config

import (
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"

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

	ctyEnvMap := make(map[string]cty.Value)
	for _, v := range os.Environ() {
		kv := strings.Split(v, "=")
		if _, ok := ctyEnvMap[kv[0]]; !ok {
			// println("addedEnv:", kv[0], kv[1])
			ctyEnvMap[kv[0]] = cty.StringVal(kv[1])
		}
	}

	for f, frontend := range config.Frontends {
		for e, endpoint := range frontend.Endpoint {
			handler := typeMap[strings.ToLower(endpoint.Backend.Type)](log)
			evalCtx := &hcl.EvalContext{Variables: map[string]cty.Value{
				"env": cty.MapVal(ctyEnvMap),
			}}
			diags := gohcl.DecodeBody(endpoint.Backend.Options, evalCtx, handler)
			if diags.HasErrors() {
				log.Fatal(diags.Error())
			}
			config.Frontends[f].Endpoint[e].Backend.instance = handler
			config.Frontends[f].Endpoint[e].Frontend = frontend // assign parent
		}
	}

	return &config
}
