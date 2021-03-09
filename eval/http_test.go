package eval_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

func Test_SetGetBody_LimitBody(t *testing.T) {
	type testCase struct {
		name    string
		limit   int64
		payload string
		wantErr error
	}

	for _, testcase := range []testCase{
		{"/w well sized limit", 1024, "content", nil},
		{"/w zero limit", 0, "01", errors.EndpointReqBodySizeExceeded},
		{"/w limit /w oversize body", 4, "12345", errors.EndpointReqBodySizeExceeded},
	} {
		t.Run(testcase.name, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(testcase.payload))

			err := eval.SetGetBody(req, testcase.limit)

			if !reflect.DeepEqual(err, testcase.wantErr) {
				subT.Errorf("Expected '%v', got: '%v'", testcase.wantErr, err)
			}
		})
	}
}
