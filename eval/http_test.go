package eval_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

func Test_SetGetBody_LimitBody(t *testing.T) {
	type testCase struct {
		name       string
		limit      int64
		payload    string
		wantErrMsg string
	}

	for _, testcase := range []testCase{
		{"/w well sized limit", 1024, "content", ""},
		{"/w zero limit", 0, "01", "body size exceeded: 0B"},
		{"/w limit /w oversize body", 4, "12345", "body size exceeded: 4B"},
	} {
		t.Run(testcase.name, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(testcase.payload))

			err := eval.SetGetBody(req, testcase.limit)
			if testcase.wantErrMsg == "" && err == nil {
				return
			}

			e := err.(errors.GoError)
			if e.GoError() != testcase.wantErrMsg {
				t.Errorf("\nWant:\t%s\nGot:\t%s", testcase.wantErrMsg, e.GoError())
			}
		})
	}
}
