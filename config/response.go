package config

import "net/http"

type Response struct {
	Headers http.Header `hcl:"headers,optional"`
}
