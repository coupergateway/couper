package producer

import "net/http"

// Result represents the producer <Result> object.
type Result struct {
	Beresp *http.Response
	Err    error
	// TODO: trace
}

// Results represents the producer <Result> channel.
type Results chan *Result
