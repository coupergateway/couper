package errors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/assets"
	"github.com/coupergateway/couper/config/request"
)

var (
	DefaultHTML *Template
	DefaultJSON *Template
)

const HeaderErrorCode = "Couper-Error"

func init() {
	var err error
	DefaultHTML, err = NewTemplate("text/html", "default.html", assets.Assets.MustOpen("error.html").Bytes(), nil)
	if err != nil {
		panic(err)
	}
	DefaultJSON, err = NewTemplate("application/json", "default.json", assets.Assets.MustOpen("error.json").Bytes(), nil)
	if err != nil {
		panic(err)
	}
}

type Template struct {
	ctxHandler http.HandlerFunc
	log        *logrus.Entry
	mime       string
	raw        []byte
	tpl        *template.Template
}

func NewTemplateFromFile(path string, logger *logrus.Entry) (*Template, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	tplFile, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	mime := "text/html"
	if strings.HasSuffix(path, ".json") {
		mime = "application/json"
	}

	_, fileName := filepath.Split(path)
	return NewTemplate(mime, fileName, tplFile, logger)
}

var setLoggerMu sync.Mutex // required for testing purposes

// SetLogger updates the default templates with the configured "daemon" logger.
func SetLogger(log *logrus.Entry) {
	setLoggerMu.Lock()
	defer setLoggerMu.Unlock()
	DefaultJSON.log = log
	DefaultHTML.log = log
}

func NewTemplate(mime, name string, src []byte, logger *logrus.Entry) (*Template, error) {
	tpl, err := template.New(name).Parse(string(src))
	if err != nil {
		return nil, err
	}

	return &Template{
		log:  logger,
		mime: mime,
		raw:  src,
		tpl:  tpl,
	}, nil
}

func (t *Template) WithContextFunc(fn http.HandlerFunc) *Template {
	tpl := *t
	tpl.ctxHandler = fn
	return &tpl

}

func (t *Template) WithError(err error) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", t.mime)

		goErr, ok := err.(GoError)
		if !ok {
			goErr = Server.With(err)
		}

		statusCode := goErr.HTTPStatus()

		*req = *req.WithContext(context.WithValue(req.Context(), request.Error, goErr))

		rw.Header().Set(HeaderErrorCode, fmt.Sprint(err.Error()))

		if t.ctxHandler != nil {
			t.ctxHandler.ServeHTTP(rw, req)
		}

		rw.WriteHeader(statusCode)

		if req.Method == http.MethodHead { // Its fine to send CT
			return
		}

		var reqID string
		if r, valOk := req.Context().Value(request.UID).(string); valOk {
			reqID = r // could be nil within (unit) test cases
		}
		data := map[string]interface{}{
			"http_status": statusCode,
			"message":     err.Error(),
			"path":        req.URL.EscapedPath(),
			"request_id":  escapeValue(t.mime, reqID),
		}
		tplErr := t.tpl.Execute(rw, data)

		// FIXME: If the fallback triggers, maybe we set
		// different/double headers on the top of this method
		// (recursive call)

		// fallback behaviour, execute internal template once
		if tplErr != nil && (t != DefaultHTML && t != DefaultJSON) {
			if !strings.Contains(t.mime, "text/html") {
				DefaultJSON.WithError(goErr).ServeHTTP(rw, req)
				return
			}
			DefaultHTML.WithError(goErr).ServeHTTP(rw, req)
		} else if tplErr != nil && t.log != nil {
			t.log.WithFields(data).Error(tplErr)
		}
	})
}

func escapeValue(mime, val string) string {
	if strings.HasPrefix(mime, "text/html") {
		return template.HTMLEscapeString(val)
	}
	return template.JSEscapeString(val)
}
