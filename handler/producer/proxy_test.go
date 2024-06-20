package producer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/handler"
	"github.com/coupergateway/couper/handler/producer"
	"github.com/coupergateway/couper/handler/transport"
	"github.com/coupergateway/couper/internal/test"
)

func Test_ProxyProduceUnexpectedStatus(t *testing.T) {
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

	backend := transport.NewBackend(&hclsyntax.Body{}, &transport.Config{Origin: origin.URL}, nil, logEntry)

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
		attr          *hclsyntax.Attribute
		reflectStatus int // send via header, reflected by origin as http status-code
		expectedErr   error
	}{
		{"/w status /w unexpected response", &hclsyntax.Attribute{
			Name: "unexpected_status",
			Expr: &hclsyntax.LiteralValueExpr{Val: toListVal(200, 304)}},
			http.StatusNotModified,
			errors.UnexpectedStatus,
		},
		{"/w status /w expected response", &hclsyntax.Attribute{
			Name: "unexpected_status",
			Expr: &hclsyntax.LiteralValueExpr{Val: toListVal(200, 304)}},
			http.StatusNotAcceptable,
			nil,
		},
	}

	for _, tt := range tests {
		content := &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{
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

		producers := []producer.Roundtrip{
			&producer.Proxy{
				Content:   content,
				Name:      "proxy",
				RoundTrip: handler.NewProxy(backend, content, false, logEntry),
			},
		}
		testNames := []string{"request", "proxy"}

		for i, rt := range producers {
			t.Run(testNames[i]+"_"+tt.name, func(t *testing.T) {

				ctx := eval.NewDefaultContext().WithClientRequest(clientRequest)

				outreq := clientRequest.WithContext(ctx)
				outreq.Header.Set("X-Status", strconv.Itoa(tt.reflectStatus))

				result := rt.Produce(outreq)

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
