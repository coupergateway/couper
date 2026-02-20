package eval_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
)

func TestSetHeader_SanitizesCRLF(t *testing.T) {
	for _, tc := range []struct {
		name      string
		input     string
		wantValue string
	}{
		{"clean value", "safe-value", "safe-value"},
		{"strips CR", "value\rwith-cr", "valuewith-cr"},
		{"strips LF", "value\nwith-lf", "valuewith-lf"},
		{"strips CRLF", "value\r\nInjected: true", "valueInjected: true"},
		{"strips null", "value\x00with-null", "valuewith-null"},
		{"strips mixed", "\r\n\x00all-removed\r\n", "all-removed"},
		{"empty stays empty", "", ""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			header := make(http.Header)
			val := cty.ObjectVal(map[string]cty.Value{
				"x-test": cty.StringVal(tc.input),
			})

			eval.SetHeader(val, header)

			got := header.Get("X-Test")
			if got != tc.wantValue {
				subT.Errorf("want %q, got %q", tc.wantValue, got)
			}
		})
	}
}

func TestSetHeader_SanitizesListValues(t *testing.T) {
	header := make(http.Header)
	val := cty.ObjectVal(map[string]cty.Value{
		"x-multi": cty.TupleVal([]cty.Value{
			cty.StringVal("clean"),
			cty.StringVal("has\r\nnewline"),
		}),
	})

	eval.SetHeader(val, header)

	values := header["X-Multi"]
	if len(values) != 2 {
		t.Fatalf("want 2 values, got %d", len(values))
	}
	if values[0] != "clean" {
		t.Errorf("values[0]: want %q, got %q", "clean", values[0])
	}
	if values[1] != "hasnewline" {
		t.Errorf("values[1]: want %q, got %q", "hasnewline", values[1])
	}
}

func TestValidatePath(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid absolute", "/api/resource", false},
		{"valid relative", "resource", false},
		{"valid wildcard", "/api/**", false},
		{"empty path", "", false},
		{"traversal at start", "/../etc/passwd", true},
		{"traversal in middle", "/api/../../etc/passwd", true},
		{"traversal bare", "..", true},
		{"traversal relative", "foo/../bar", true},
		{"encoded traversal", "/api/%2e%2e/secret", true},
		{"encoded traversal uppercase", "/api/%2E%2E/secret", true},
		{"single dot ok", "/api/./resource", false},
		{"dots in filename ok", "/api/file..name", false},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			err := eval.ValidatePath(tc.path, "test")
			if (err != nil) != tc.wantErr {
				subT.Errorf("ValidatePath(%q): got err=%v, wantErr=%v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func Test_SetGetBody_LimitBody(t *testing.T) {
	type testCase struct {
		name       string
		limit      int64
		payload    string
		wantErrMsg string
	}

	for _, testcase := range []testCase{
		{"/w well sized limit", 1024, "content", ""},
		{"/w zero limit", 0, "01", "client request error: body size exceeded: 0B"},
		{"/w limit /w oversize body", 4, "12345", "client request error: body size exceeded: 4B"},
	} {
		t.Run(testcase.name, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(testcase.payload))

			err := eval.SetGetBody(req, buffer.Request, testcase.limit)
			if testcase.wantErrMsg == "" && err == nil {
				return
			}

			e := err.(errors.GoError)
			if e.LogError() != testcase.wantErrMsg {
				t.Errorf("\nWant:\t%s\nGot:\t%s", testcase.wantErrMsg, e.LogError())
			}
		})
	}
}
