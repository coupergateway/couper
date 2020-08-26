package runtime

import (
	"net/http"
	"reflect"
	"regexp"
	"testing"
)

func TestNewRoute(t *testing.T) {
	noopHandlerFn := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	type args struct {
		pattern string
		handler http.Handler
	}

	tests := []struct {
		name    string
		args    args
		want    *Route
		wantErr bool
	}{
		{"empty pattern", args{"", nil}, nil, true},
		{"missing root slash", args{"path", nil}, nil, true},
		{"missing handler", args{"/", nil}, nil, true},
		{"path: /", args{"/", noopHandlerFn}, &Route{pattern: "/", matcher: regexp.MustCompile("^/$"), handler: noopHandlerFn}, false},
		{"path: /sub/", args{"/sub/", noopHandlerFn}, &Route{pattern: "/sub/", matcher: regexp.MustCompile("^/sub/$"), handler: noopHandlerFn}, false},
		{"path: /**", args{"/**", noopHandlerFn}, &Route{pattern: "/**", matcher: regexp.MustCompile("^/?(.*)"), handler: noopHandlerFn}, false},
		{"path: /sub", args{"/sub/**", noopHandlerFn}, &Route{pattern: "/sub/**", matcher: regexp.MustCompile("^/sub/?(.*)"), handler: noopHandlerFn}, false},
		{"path: /sub/**", args{"/sub/**", noopHandlerFn}, &Route{pattern: "/sub/**", matcher: regexp.MustCompile("^/sub/?(.*)"), handler: noopHandlerFn}, false},
		{"path: /sub/**/foo/", args{"/sub/**/foo/", noopHandlerFn}, nil, true},
		{"path: /sub/**/foo/**", args{"/sub/**/foo/**", noopHandlerFn}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRoute(tt.args.pattern, tt.args.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if tt.wantErr && err != nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("NewRoute() got = %v, want %v", got, tt.want)
				}
				return
			}

			if reflect.ValueOf(got.handler).Pointer() != reflect.ValueOf(tt.want.handler).Pointer() {
				t.Errorf("NewRoute() got = %v, want %v", got.handler, tt.want.handler)
			}

			if got.matcher.String() != tt.want.matcher.String() {
				t.Errorf("NewRoute() got = %v, want %v", got.matcher.String(), tt.want.matcher.String())
			}
		})
	}
}
