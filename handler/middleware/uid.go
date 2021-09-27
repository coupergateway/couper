package middleware

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

type UID struct {
	conf           *config.Settings
	devProxyHeader string
	generate       UIDFunc
	handler        http.Handler
}

func NewUIDHandler(conf *config.Settings, devProxy string) Next {
	return func(handler http.Handler) http.Handler {
		return &UID{
			conf:           conf,
			devProxyHeader: devProxy,
			generate:       NewUIDFunc(conf.RequestIDFormat),
			handler:        handler,
		}
	}
}

// ServeHTTP generates a unique request-id and add this id to the request context and
// at least the response header even on error case.
func (u *UID) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	uid, err := u.newUID(req.Header)

	*req = *req.WithContext(context.WithValue(req.Context(), request.UID, uid))

	if u.conf.RequestIDClientHeader != "" {
		rw.Header().Set(u.conf.RequestIDClientHeader, uid)
	}

	if err != nil {
		errors.DefaultHTML.ServeError(errors.ClientRequest.With(err)).ServeHTTP(rw, req)
		return
	}

	if u.conf.RequestIDBackendHeader != "" {
		req.Header.Set(u.conf.RequestIDBackendHeader, uid)
	}

	u.handler.ServeHTTP(rw, req)
}

func (u *UID) newUID(header http.Header) (string, error) {
	if u.conf.RequestIDAcceptFromHeader != "" {
		if v := header.Get(u.conf.RequestIDAcceptFromHeader); v != "" {
			if !regexUID.MatchString(v) {
				return u.generate(), errors.ClientRequest.
					Messagef("invalid request-id header value: %s: %s", u.conf.RequestIDAcceptFromHeader, v)
			}

			return v, nil
		}
	} else if httpsDevProxyID := header.Get(u.devProxyHeader); httpsDevProxyID != "" {
		header.Del(u.devProxyHeader)
		return httpsDevProxyID, nil
	}
	return u.generate(), nil
}

// UIDFunc wraps different unique id implementations.
type UIDFunc func() string

func NewUIDFunc(requestIDFormat string) UIDFunc {
	var fn UIDFunc
	if requestIDFormat == "uuid4" {
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
