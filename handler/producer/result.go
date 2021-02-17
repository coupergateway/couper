package producer

import "net/http"

type Result struct {
	Beresp *http.Response
	Err    error
	// TODO: trace
}

type Results chan *Result
