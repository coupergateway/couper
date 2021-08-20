package server

import (
	"context"
	"net/http"
	"regexp"

	"github.com/rs/xid"
	uuid "github.com/satori/go.uuid"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var regexUID = regexp.MustCompile(`^[a-zA-Z0-9@=/+-]{12,64}$`)

// uidFunc wraps different unique id implementations.
type uidFunc func() string

func newUIDFunc(settings *config.Settings) uidFunc {
	var fn uidFunc
	if settings.RequestIDFormat == "uuid4" {
		fn = func() string {
			return uuid.NewV4().String()
		}
	} else {
		fn = func() string {
			return xid.New().String()
		}
	}
	return fn
}

func (s *HTTPServer) getUID(req *http.Request) (string, error) {
	uid := s.uidFn()

	if afh := s.settings.RequestIDAcceptFromHeader; afh != "" {
		h := req.Header.Get(afh)
		if h == "" {
			return uid, nil
		}

		if !regexUID.MatchString(h) {
			return uid, errors.ClientRequest.Messagef("invalid request-ID %q given in header %q", h, afh)
		}

		uid = h
	} else if httpsDevProxyID := req.Header.Get(httpsDevProxyIDField); httpsDevProxyID != "" {
		uid = httpsDevProxyID
		req.Header.Del(httpsDevProxyIDField)
	}

	return uid, nil
}

func (s *HTTPServer) setUID(rw http.ResponseWriter, req *http.Request) error {
	uid, err := s.getUID(req)

	ctx := context.WithValue(req.Context(), request.UID, uid)

	if h := s.settings.RequestIDBackendHeader; h != "" {
		req.Header.Set(h, uid)
	}

	if h := s.settings.RequestIDClientHeader; h != "" {
		rw.Header().Set(h, uid)
	}

	*req = *req.WithContext(ctx)

	return err
}
