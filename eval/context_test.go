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
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/request"
)

func TestNewHTTPContext(t *testing.T) {
	t.Skip("TODO")
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
		ReqHeader cty.Value `hcl:"headers,optional"`
		Method    cty.Value `hcl:"method,optional"`
		QueryVar  cty.Value `hcl:"query,optional"`
		PostVar   cty.Value `hcl:"post,optional"`
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
		post = req.post.user
		method = req.method
`, EvalTestContext{PostVar: cty.ListVal([]cty.Value{cty.StringVal("hans")}), Method: cty.ListVal([]cty.Value{cty.StringVal(http.MethodPost)})}},
		{"Variables / Query", http.MethodGet, nil, "?name=peter", baseCtx, `
				query = req.query.name
				method = req.method
				headers = req.headers.user-agent
		`, EvalTestContext{QueryVar: cty.StringVal("peter"), Method: cty.StringVal(http.MethodGet), ReqHeader: cty.ListVal([]cty.Value{cty.StringVal("test/v1")})}},
		{"Variables / Headers", http.MethodGet, nil, "", baseCtx, `
				headers = req.headers.user-agent
				method = req.method
		`, EvalTestContext{ReqHeader: cty.StringVal("test/v1"), Method: cty.StringVal(http.MethodGet)}},
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

			got := NewHTTPContext(tt.baseCtx, BufferRequest, req, bereq, beresp)
			got.Functions = nil // we are not interested in a functions test

			var hclResult EvalTestContext
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), got, &hclResult)
			if err != nil {
				t.Fatal(err)
			}

			value := reflect.ValueOf(hclResult)
			for i := 0; i < value.NumField(); i++ {
				tag := reflect.TypeOf(hclResult).Field(i).Tag.Get("hcl")
				name := strings.Split(tag, ",")[0]

				want := reflect.ValueOf(tt.want).Field(i).Interface().(cty.Value)
				if want.IsNull() {
					continue
				}

				gotVar := got.Variables["req"].GetAttr(name)
				if !want.RawEquals(gotVar) {
					t.Errorf("cty.Value equals()\ngot:\t%v\nwant:\t%v", gotVar, want)
				}
			}
		})
	}
}
