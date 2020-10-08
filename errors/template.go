package errors

import (
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"text/template"

	"github.com/avenga/couper/assets"
	"github.com/avenga/couper/config/request"
)

var (
	DefaultHTML *Template
	DefaultJSON *Template
)

const HeaderErrorCode = "Couper-Error"

func init() {
	var err error
	DefaultHTML, err = NewTemplate("text/html", assets.Assets.MustOpen("error.html").Bytes())
	if err != nil {
		panic(err)
	}
	DefaultJSON, err = NewTemplate("application/json", assets.Assets.MustOpen("error.json").Bytes())
	if err != nil {
		panic(err)
	}
}

type ErrorTemplate interface {
	Template() *Template
}

type Template struct {
	raw  []byte
	mime string
	tpl  *template.Template
}

func NewTemplateFromFile(path string) (*Template, error) {
	tplFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewTemplate(mime.TypeByExtension(path), tplFile)
}

func NewTemplate(mime string, src []byte) (*Template, error) {
	tpl, err := template.New("").Parse(string(src))
	if err != nil {
		return nil, err
	}

	return &Template{
		mime: mime,
		raw:  src,
		tpl:  tpl,
	}, nil
}

func (t *Template) ServeError(err error) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", t.mime)

		errCode, ok := err.(Code)
		if !ok {
			errCode = Server
		}

		SetHeader(rw, errCode)

		status := httpStatus(errCode)
		rw.WriteHeader(status)

		if req.Method == http.MethodHead { // Its fine to send CT
			return
		}

		var reqID string
		if r, ok := req.Context().Value(request.UID).(string); ok {
			reqID = r // could be nil within (unit) test cases
		}
		data := map[string]interface{}{
			"http_status": status,
			"message":     err.Error(),
			"error_code":  int(errCode),
			"path":        req.URL.EscapedPath(),
			"request_id":  escapeValue(t.mime, reqID),
		}
		err := t.tpl.Execute(rw, data)

		// FIXME: If the fallback triggers, maybe we set
		// different/double headers on the top of this method
		// (recursive call)

		// fallback behaviour, execute internal template once
		if err != nil && (t != DefaultHTML && t != DefaultJSON) {
			if !strings.Contains(t.mime, "text/html") {
				DefaultJSON.ServeError(errCode).ServeHTTP(rw, req)
				return
			}
			DefaultJSON.ServeError(errCode).ServeHTTP(rw, req)
		} else if err != nil {
			panic(err)
		}
	})
}

func escapeValue(mime, val string) string {
	if strings.HasPrefix(mime, "text/html") {
		return template.HTMLEscapeString(val)
	}
	return template.JSEscapeString(val)
}
