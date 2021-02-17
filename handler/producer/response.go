package producer

import (
	"io"
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

// Response represents the producer <Response> object.
type Response struct {
	Body      io.Reader
	Context   hcl.Body
	Header    http.Header
	Reference string
	Status    int
}
