package producer

import (
	"github.com/hashicorp/hcl/v2"
)

// Response represents the producer <Response> object.
type Response struct {
	Context hcl.Body
}
