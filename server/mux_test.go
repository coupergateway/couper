package server_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	rs "github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/server"
)

func TestMux_FindHandler_PathParamContext(t *testing.T) {
	type noContentHandler http.Handler
	var noContent noContentHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	serverOptions, _ := rs.NewServerOptions(nil, nil)

	testOptions := &runtime.MuxOptions{
		EndpointRoutes: map[string]http.Handler{
			"/without":              http.NotFoundHandler(),
			"/with/{my}/parameter":  noContent,
			"/{section}/cms":        noContent,
			"/a/{b}":                noContent,
			"/{b}/a":                noContent,
			"/{section}/{project}":  noContent,
			"/project/{project}/**": noContent,
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
		{" /w 2nd path param", newReq("/a/c"), noContent, request.PathParameter{
			"b": "c",
		}, ""},
		{" /w two path param", newReq("/foo/bar"), noContent, request.PathParameter{
			"section": "foo",
			"project": "bar",
		}, ""},
		{" /w path param and expWildcard", newReq("/project/foo/bar/ha"), noContent, request.PathParameter{
			"project": "foo",
		}, "bar/ha"},
		{" /w non existing path", newReq("/foo/{bar}/123"), errors.DefaultJSON.ServeError(errors.RouteNotFound), nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := server.NewMux(testOptions)

			if got := mux.FindHandler(tt.req); reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindHandler() = %v, want %v", got, tt.want)
			}

			paramCtx, ok := tt.req.Context().Value(request.PathParams).(request.PathParameter)
			if !ok {
				if tt.expParams == nil {
					return
				}
				t.Errorf("Expected path parameters")
				return
			}

			if !reflect.DeepEqual(paramCtx, tt.expParams) {
				t.Errorf("Path parameter context: %#v, want: %#v", paramCtx, tt.expParams)
			}

			wildcardCtx, ok := tt.req.Context().Value(request.Wildcard).(string)
			if !ok {
				if tt.expWildcard == "" {
					return
				}
				t.Fatal("Expected expWildcard value")
			}

			if wildcardCtx != tt.expWildcard {
				t.Errorf("Wildcard context: %q, want: %q", wildcardCtx, tt.expWildcard)
			}
		})
	}
}
