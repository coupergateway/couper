package config

import (
	"net/http"
)

type Api struct {
	AccessControl        []string    `hcl:"access_control,optional"`
	DisableAccessControl []string    `hcl:"disable_access_control,optional"`
	BasePath             string      `hcl:"base_path,optional"`
	Endpoint             []*Endpoint `hcl:"endpoint,block"`
	PathHandler          PathHandler
}

type PathHandler map[*Endpoint]http.Handler
