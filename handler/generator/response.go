package generator

import (
	"io"
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

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
