package test

import "net/http"

type Header map[string]string

func (h Header) Set(req *http.Request) Header {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	for k, v := range h {
		req.Header.Set(k, v)
	}
	return h
}
