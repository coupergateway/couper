package config

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"
)

type Api struct {
	BasePath    string      `hcl:"base_path,optional"`
	Backend     []*Backend  `hcl:"backend,block"`
	Endpoint    []*Endpoint `hcl:"endpoint,block"`
	PathHandler PathHandler
}

type PathHandler map[*Endpoint]http.Handler

type HandlerWrapper struct {
	log     *logrus.Entry
	acs     []*Jwt
	handler http.Handler
}

func NewHandlerWrapper(log *logrus.Entry, acs []*Jwt, handler http.Handler) http.Handler {
	handlerWrapper := &HandlerWrapper{log: log, acs: acs, handler: handler}
	for j := range acs {
		acs[j].Init(log)
	}
	return handlerWrapper
}

func (hw *HandlerWrapper) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, jwt := range hw.acs {
		if !jwt.Check(req) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	hw.handler.ServeHTTP(rw, req)
}

func (api *Api) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(api)
	if !inline {
		return schema
	}
	// backend, remove 2nd label for inline usage
	for i, block := range schema.Blocks {
		if block.Type == "backend" && len(block.LabelNames) > 1 {
			schema.Blocks[i].LabelNames = block.LabelNames[:1]
		}
	}
	return schema
}
