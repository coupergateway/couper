package middleware

import (
	"context"
	"net/http"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/server/writer"
)

func NewRecordHandler(secureCookies string) Next {
	return func(handler http.Handler) *NextHandler {
		return NewHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			gw := writer.NewGzipWriter(rw, req.Header)
			w := writer.NewResponseWriter(gw, secureCookies)

			// This defer closes the GZ writer but more important is triggering our own buffer logic in all cases
			// for this writer to prevent the 200 OK status fallback (http.ResponseWriter) and an empty response body.
			defer func() {
				select { // do not close on cancel since we may have nothing to write and the client may be gone anyways.
				case <-req.Context().Done():
					return
				default:
					_ = gw.Close()
				}
			}()

			ctx := context.WithValue(req.Context(), request.ResponseWriter, w)
			*req = *req.WithContext(ctx)
			handler.ServeHTTP(w, req)
		}), handler)
	}
}
