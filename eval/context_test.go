package eval

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"

	"github.com/avenga/couper/config/request"
)

func TestNewHTTPContext(t *testing.T) {
	newBeresp := func(br *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "OK",
			Request:    br,
			Proto:      br.Proto,
			ProtoMajor: br.ProtoMajor,
			ProtoMinor: br.ProtoMinor,
		}
	}

	type EvalTestContext struct {
		ReqHeader string `hcl:"header,optional"`
		Method    string `hcl:"method,optional"`
		QueryVar  string `hcl:"query_var,optional"`
		PostVar   string `hcl:"post_var,optional"`
	}

	baseCtx := NewENVContext(nil)

	tests := []struct {
		name      string
		reqMethod string
		body      io.Reader
		query     string
		baseCtx   *hcl.EvalContext
		hcl       string
		want      EvalTestContext
	}{
		{"Variables / POST", http.MethodPost, strings.NewReader(`user=hans`), "", baseCtx, `
		post_var = req.post.user
		method = req.method
`, EvalTestContext{PostVar: "hans", Method: http.MethodPost}},
		{"Variables / Query", http.MethodGet, nil, "?name=peter", baseCtx, `
		query_var = req.query.name
		method = req.method
		header = req.headers.user-agent
`, EvalTestContext{QueryVar: "peter", Method: http.MethodGet, ReqHeader: "test/v1"}},
		{"Variables / Headers", http.MethodGet, nil, "", baseCtx, `
		header = req.headers.user-agent
`, EvalTestContext{ReqHeader: "test/v1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.reqMethod, "https://couper.io/"+tt.query, tt.body)
			*req = *req.Clone(context.WithValue(req.Context(), request.Endpoint, "couper-proxy"))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			req.Header.Set("User-Agent", "test/v1")

			bereq := req.Clone(context.Background())
			beresp := newBeresp(bereq)

			got := NewHTTPContext(tt.baseCtx, true, req, bereq, beresp)
			got.Functions = nil // we are not interested in a functions test

			var hclResult EvalTestContext
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), got, &hclResult)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(hclResult, tt.want) {
				t.Errorf("NewHTTPContext()\ngot:\t%v\nwant:\t%v", hclResult, tt.want)
			}
		})
	}
}
