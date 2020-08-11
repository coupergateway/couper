package handler

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"go.avenga.cloud/couper/gateway/assets"
	"go.avenga.cloud/couper/gateway/utils"
)

const RequestIDKey = "requestID"

var _ http.Handler = &ServingError{}

// ServingError represents a ServingError object
type ServingError struct {
	Asset      *assets.AssetFile
	Code       int
	HTTPStatus int
	Message    string
}

func NewErrorHandler(asset *assets.AssetFile, code, status int) *ServingError {
	return &ServingError{
		Asset:      asset,
		Code:       code,
		HTTPStatus: status,
		Message:    utils.GetErrorMessage(code),
	}
}

func (s *ServingError) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if s.Asset == nil || s.Asset.Tpl() == nil {
		rw.WriteHeader(s.HTTPStatus)
		return
	}

	confBytes := &bytes.Buffer{}
	err := s.Asset.Tpl().Execute(confBytes, map[string]interface{}{
		"http_status": s.HTTPStatus,
		"message":     s.escapeValue(s.Message),
		"error_code":  s.Code,
		"path":        req.URL.EscapedPath(),
		"request_id":  s.escapeValue(req.Context().Value(RequestIDKey).(string)),
	})
	if err != nil {
		rw.WriteHeader(s.HTTPStatus)
		return
	}

	if req.Method != http.MethodHead {
		if ct := s.Asset.CT(); ct != "" {
			rw.Header().Set("Content-Type", ct)
		}
		// FIXME: The asset-size is changed after replacements
		// rw.Header().Set("Content-Length", s.Asset.Size())
	}

	rw.WriteHeader(s.HTTPStatus)

	// TODO: gzip, br?
	if req.Method != "HEAD" {
		rw.Write(confBytes.Bytes())
	}
}

func (s *ServingError) escapeValue(v string) string {
	if strings.HasPrefix(s.Asset.CT(), "text/html") {
		return template.HTMLEscapeString(v)
	}

	return template.JSEscapeString(v)
}

func (s *ServingError) String() string {
	return "Error"
}
