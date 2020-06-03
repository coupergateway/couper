package config

import "net/http"

type Request struct {
	Headers http.Header `hcl:"headers,optional"`
}
