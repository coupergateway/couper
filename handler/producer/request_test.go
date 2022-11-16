package producer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/test"
)

func Test_ProduceExpectedStatus(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		s, err := strconv.Atoi(req.Header.Get("X-Status"))
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(s)
	}))
	defer origin.Close()

	logger, _ := test.NewLogger()
	logEntry := logger.WithContext(context.Background())

	backend := transport.NewBackend(hcl.EmptyBody(), &transport.Config{Origin: origin.URL}, nil, logEntry)

	clientRequest, _ := http.NewRequest(http.MethodGet, "http://couper.local", nil)

	toListVal := func(numbers ...int64) cty.Value {
		var list []cty.Value
		for _, n := range numbers {
			list = append(list, cty.NumberIntVal(n))
		}
		return cty.ListVal(list)
	}

	tests := []struct {
		name          string
		attr          *hcl.Attribute
		reflectStatus int // send via header, reflected by origin as http status-code
		expectedErr   error
	}{
		{"/wo status", nil, http.StatusNoContent, nil},
		{"/w status /w unexpected response", &hcl.Attribute{
			Name: "expected_status",
			Expr: &hclsyntax.LiteralValueExpr{Val: toListVal(200, 304)}},
			http.StatusNotAcceptable,
			errors.UnexpectedStatus,
		},
		{"/w status /w expected response", &hcl.Attribute{
			Name: "expected_status",
			Expr: &hclsyntax.LiteralValueExpr{Val: toListVal(200, 304)}},
			http.StatusNotModified,
			nil,
		},
	}

	for _, tt := range tests {
		content := &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"url": {Name: "url", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(origin.URL)}},
			// Since request will not proxy our dynamic client-request header value, we will add a headers attr here.
			// There is no validation, so this also applies to proxy (unused)
			"set_request_headers": {Name: "set_request_headers", Expr: &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{
					{
						KeyExpr:   &hclsyntax.LiteralValueExpr{Val: cty.StringVal("X-Status")},
						ValueExpr: &hclsyntax.LiteralValueExpr{Val: cty.NumberIntVal(int64(tt.reflectStatus))},
					},
				},
			}},
		}}
		if tt.attr != nil {
			content.Attributes[tt.attr.Name] = tt.attr
		}

		requests := producer.Requests{&producer.Request{
			Backend: backend,
			Context: hclbody.New(content),
			Name:    "request",
		}}

		proxies := producer.Proxies{&producer.Proxy{
			Content:   hclbody.New(content),
			Name:      "proxy",
			RoundTrip: handler.NewProxy(backend, hclbody.New(content), logEntry),
		}}

		testNames := []string{"request", "proxy"}

		for i, rt := range []producer.Roundtrip{requests, proxies} {
			t.Run(testNames[i]+"_"+tt.name, func(t *testing.T) {

				ctx := eval.NewDefaultContext().WithClientRequest(clientRequest)

				outreq := clientRequest.WithContext(ctx)
				outreq.Header.Set("X-Status", strconv.Itoa(tt.reflectStatus))

				result := <-rt.Produce(outreq, make(map[string][]chan *producer.Result))

				if !errors.Equals(tt.expectedErr, result.Err) {
					t.Fatalf("expected error: %v, got %v", tt.expectedErr, result.Err)
				}

				if result.Beresp == nil {
					t.Fatal("expected a backend response")
				}
			})
		}
	}
}
