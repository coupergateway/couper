package server

import (
	"context"
	"net/http"
	"regexp"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var regexUID = regexp.MustCompile(`^[a-zA-Z0-9@=/+-]{12,64}$`)

func getUID(s *HTTPServer, req *http.Request) (string, error) {
	uid := s.uidFn()

	if afh := s.settings.RequestIDAcceptFromHeader; afh != "" {
		h := req.Header.Get(afh)

		if !regexUID.MatchString(h) {
			return uid, errors.ClientRequest.Messagef("invalid request-ID %q given in header %q", h, afh)
		}

		uid = h
	}

	return uid, nil
}

func setUID(s *HTTPServer, rw http.ResponseWriter, req *http.Request) error {
	uid, err := getUID(s, req)

	ctx := context.WithValue(req.Context(), request.UID, uid)
	ctx = context.WithValue(ctx, request.LogEntry, s.log.WithField("uid", uid))

	if h := s.settings.RequestIDBackendHeader; h != "" {
		req.Header.Set(h, uid)
	}
	if h := s.settings.RequestIDClientHeader; h != "" {
		rw.Header().Set(h, uid)
	}

	*req = *req.WithContext(ctx)

	return err
}
