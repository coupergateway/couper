package server

import (
	"net/http"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/utils"
)

// normalizeRequestPath rewrites req.URL to the canonical form of its path so that
// routing, access control and the upstream request agree on one resource.
func normalizeRequestPath(req *http.Request) *errors.Error {
	if err := utils.NormalizeURLPath(req.URL); err != nil {
		return errors.ClientRequest.With(err)
	}

	// Only an encoded slash can leave an empty or dot segment in the decoded path.
	if utils.HasEmptyOrDotSegment(req.URL.Path) {
		return errors.ClientRequest.Message("encoded slash forms an empty or dot segment")
	}

	return nil
}
