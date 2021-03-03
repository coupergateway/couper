package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/internal/test"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestEndpoint_SetGetBody_LimitBody_Roundtrip(t *testing.T) {
	type testCase struct {
		name    string
		limit   string
		payload string
		wantErr error
	}

	for _, testcase := range []testCase{
		{"/w well sized limit", "12MiB", "content", nil},
		{"/w zero limit", "0", "01", errors.APIReqBodySizeExceeded},
		{"/w limit /w oversize body", "4B", "12345", errors.APIReqBodySizeExceeded},
	} {
		t.Run(testcase.name, func(subT *testing.T) {
			helper := test.New(subT)

			logger, _ := logrustest.NewNullLogger()
			log := logger.WithContext(context.Background())

			bodyLimit, _ := handler.ParseBodyLimit(testcase.limit)
			epHandler := handler.NewEndpoint(&handler.EndpointOptions{
				Context:      helper.NewProxyContext("set_request_headers = { x = req.post }"),
				ReqBodyLimit: bodyLimit,
			}, eval.NewENVContext(nil), log, nil, nil, nil)

			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(testcase.payload))
			err := epHandler.SetGetBody(req)
			if !reflect.DeepEqual(err, testcase.wantErr) {
				subT.Errorf("Expected '%v', got: '%v'", testcase.wantErr, err)
			}
		})
	}
}
