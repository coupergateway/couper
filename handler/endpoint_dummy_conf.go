package handler

import (
	"io"
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

type Request struct {
	Backend *Backend
	Body    io.Reader
	Context hcl.Body
	// Dispatch bool
	Method string
	Name   string // label
	URL    string
}

type Response struct {
	Body      io.Reader
	Context   hcl.Body
	Header    http.Header
	Reference string
	Status    int
}

type Redirect struct {
	Response
}
