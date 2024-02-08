package server_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/config/runtime"
	rs "github.com/coupergateway/couper/config/runtime/server"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/server"
)

func TestSortPathPatterns(t *testing.T) {
	pathPatterns := []string{
		"/a/b/c",
		"/a/b/{c}",
		"/**",
		"/{a}/b/{c}",
		"/a/b/**",
		"/a/{b}/c",
		"/",
		"/{a}/{b}/**",
		"/a/{b}",
		"/a/**",
		"/a/{b}/{c}",
		"/{a}",
		"/{a}/b/c",
		"/{a}/b/**",
		"/{a}/{b}/c",
		"/{a}/**",
		"/{a}/{b}/{c}",
		"/a/b",
		"/{a}/b",
		"/{a}/{b}",
		"/a",
	}
	server.SortPathPatterns(pathPatterns)
	expectedSortedPathPatterns := []string{
		"/a/b/c",
		"/a/b/{c}",
		"/a/{b}/c",
		"/a/{b}/{c}",
		"/{a}/b/c",
		"/{a}/b/{c}",
		"/{a}/{b}/c",
		"/{a}/{b}/{c}",
		"/a/b",
		"/a/{b}",
		"/{a}/b",
		"/{a}/{b}",
		"/a",
		"/{a}",
		"/",
		"/a/b/**",
		"/{a}/b/**",
		"/{a}/{b}/**",
		"/a/**",
		"/{a}/**",
		"/**",
	}
	if !reflect.DeepEqual(expectedSortedPathPatterns, pathPatterns) {
		t.Errorf("exp: %v\ngot:%v", expectedSortedPathPatterns, pathPatterns)
	}
}

func TestMux_FindHandler_PathParamContext(t *testing.T) {
	type noContentHandler http.Handler
	var noContent noContentHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	serverOptions, _ := rs.NewServerOptions(nil, nil)
	serverOptions.FilesBasePaths = []string{"/"}

	testOptions := &runtime.MuxOptions{
		EndpointRoutes: map[string]http.Handler{
			"/without":              http.NotFoundHandler(),
			"/with/{my}/parameter":  noContent,
			"/{section}/cms":        noContent,
			"/a/{b}":                noContent,
			"/{b}/a":                noContent,
			"/{section}/{project}":  noContent,
			"/project/{project}/**": noContent,
			"/w/c/{param}/**":       noContent,
			"/prefix/**":            noContent,
		},
		FileRoutes: map[string]http.Handler{
			"/htdocs/test.html": noContent,
		},
		ServerOptions: serverOptions,
	}

	newReq := func(path string) *http.Request {
		return httptest.NewRequest(http.MethodGet, path, nil)
	}

	tests := []struct {
		name        string
		req         *http.Request
		want        http.Handler
		expParams   request.PathParameter
		expWildcard string
	}{
		{" /wo path params", newReq("/without"), http.NotFoundHandler(), request.PathParameter{}, ""},
		{" /w path params", newReq("/with/my123/parameter"), noContent, request.PathParameter{
			"my": "my123",
		}, ""},
		{" /w 1st path param #1", newReq("/with/cms"), noContent, request.PathParameter{
			"section": "with",
		}, ""},
		{" /w 1st path param #2", newReq("/c/a"), noContent, request.PathParameter{
			"b": "c",
		}, ""},
		{" w/o 1nd path param", newReq("//a"), http.NotFoundHandler(), nil, ""},
		{" /w 2nd path param", newReq("/a/c"), noContent, request.PathParameter{
			"b": "c",
		}, ""},
		{" w/o 2nd path param", newReq("/a"), http.NotFoundHandler(), nil, ""},
		{" w/o 2nd path param", newReq("/a/"), http.NotFoundHandler(), nil, ""},
		{" w/o 2nd path param", newReq("/a//"), http.NotFoundHandler(), nil, ""},
		{" /w two path param", newReq("/foo/bar"), noContent, request.PathParameter{
			"section": "foo",
			"project": "bar",
		}, ""},
		{" w/o two path params", newReq("//"), http.NotFoundHandler(), nil, ""},
		{" /w path param and expWildcard", newReq("/project/foo/bar/ha"), noContent, request.PathParameter{
			"project": "foo",
		}, "bar/ha"},
		{" /w non existing path", newReq("/foo/{bar}/123"), errors.DefaultJSON.WithError(errors.RouteNotFound), nil, ""},
		{" files", newReq("/htdocs/test.html"), http.NotFoundHandler(), request.PathParameter{
			"section": "htdocs",
			"project": "test.html",
		}, ""},
		{" w/ path param and wildcard", newReq("/w/c/my-param/wild/ca/rd"), noContent, request.PathParameter{
			"param": "my-param",
		}, "wild/ca/rd"},
		{" w/ path param, w/o wildcard", newReq("/w/c/my-param"), noContent, request.PathParameter{
			"param": "my-param",
		}, ""},
		{" w/ path param, trailing /, w/o wildcard", newReq("/w/c/my-param/"), noContent, request.PathParameter{
			"param": "my-param",
		}, ""},
		{" w/o path param, w/ wildcard", newReq("/w/c//wild/card"), http.NotFoundHandler(), nil, ""},
		{" w/o path param, w/o wildcard", newReq("/w/c"), noContent, request.PathParameter{
			"section": "w",
			"project": "c",
		}, ""},
		{" w/o path param, w/o wildcard, single /", newReq("/w/c/"), noContent, request.PathParameter{
			"section": "w",
			"project": "c",
		}, ""},
		{" w/o path param, w/o wildcard, double /", newReq("/w/c//"), http.NotFoundHandler(), nil, ""},
		{" w/o path param, w/o wildcard, triple /", newReq("/w/c///"), http.NotFoundHandler(), nil, ""},
		{"** not just concatenating", newReq("/prefixandsomethingelse"), http.NotFoundHandler(), nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			mux := server.NewMux(testOptions)
			mux.RegisterConfigured()

			if got := mux.FindHandler(tt.req); reflect.DeepEqual(got, tt.want) {
				subT.Errorf("FindHandler() = %v, want %v", got, tt.want)
			}

			paramCtx, ok := tt.req.Context().Value(request.PathParams).(request.PathParameter)
			if !ok {
				if tt.expParams == nil {
					return
				}
				subT.Fatalf("Expected path parameters")
			}

			if !reflect.DeepEqual(paramCtx, tt.expParams) {
				subT.Errorf("Path parameter context: %#v, want: %#v", paramCtx, tt.expParams)
			}

			wildcardCtx, ok := tt.req.Context().Value(request.Wildcard).(string)
			if !ok {
				if tt.expWildcard == "" {
					return
				}
				subT.Fatal("Expected expWildcard value")
			}

			if wildcardCtx != tt.expWildcard {
				subT.Errorf("Wildcard context: %q, want: %q", wildcardCtx, tt.expWildcard)
			}
		})
	}
}
