package server

import (
	"net/http"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/utils"
)

// normalizeRequestPath rewrites req.URL to the canonical form of its path so that
// routing, access control and the upstream request agree on one resource.
// On error req.URL stays as sent, so error templates and logs show that path.
func normalizeRequestPath(req *http.Request) *errors.Error {
	// Go parses a request target like "http:foo" into Opaque. It has no path to
	// normalize, and RequestURI would pass it to the upstream as is.
	if req.URL.Opaque != "" {
		return errors.ClientRequest.Message("opaque request target not allowed")
	}

	u := *req.URL
	if err := utils.NormalizeURLPath(&u); err != nil {
		return errors.ClientRequest.With(err)
	}

	// Only an encoded slash can leave an empty or dot segment in the decoded path.
	if utils.HasEmptyOrDotSegment(u.Path) {
		return errors.ClientRequest.Message("encoded slash forms an empty or dot segment")
	}

	*req.URL = u
	return nil
}
