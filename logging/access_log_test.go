package logging_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/server/writer"
)

func TestAccessLog_ServeHTTP(t *testing.T) {
	logger, hook := test.NewLogger()

	defer func() {
		if t.Failed() {
			for _, e := range hook.AllEntries() {
				t.Log(e.String())
			}
		}
	}()

	accessLog := logging.NewAccessLog(logging.DefaultConfig, logger)

	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	type testcase struct {
		description string
		url         string
		expFields   logrus.Fields
	}

	testcases := []testcase{
		{
			description: "pathless url",
			url:         "http://test.com",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"path": "",
				},
			},
		},
		{
			description: "proto https",
			url:         "https://example.com",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"tls":   true,
					"proto": "https",
				},
			},
		},
		{
			description: "proto http",
			url:         "http://example.com",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"tls":   false,
					"proto": "http",
				},
			},
		},
		{
			description: "method status constant",
			url:         "http://example.com",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"method": http.MethodGet,
					"status": 204,
				},
				"method": http.MethodGet,
				"status": 204,
			},
		},
		{
			description: "explicit port",
			url:         "http://localhost:8080/",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"host":   "localhost",
					"origin": "localhost:8080",
					"path":   "/",
				},
				"port": "8080",
			},
		},
		{
			description: "implicit port https",
			url:         "https://couper.io/",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"host":   "couper.io",
					"origin": "couper.io",
					"path":   "/",
				},
				"port": "443",
			},
		},
		{
			description: "implicit port http",
			url:         "http://example.com",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"host":   "example.com",
					"origin": "example.com",
					"path":   "",
				},
				"port": "80",
			},
		},
		{
			description: "required request fields",
			url:         "http://localhost:8080/test",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"tls":    false,
					"host":   "localhost",
					"origin": "localhost:8080",
					"path":   "/test",
					"method": http.MethodGet,
					"status": 204,
					"proto":  "http",
				},
				"method": http.MethodGet,
				"port":   "8080",
				"uid":    "veryRandom123",
				"api":    "myapi",
				"status": 204,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(subT *testing.T) {

			hook.Reset()

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)

			ctx := context.Background()
			ctx = context.WithValue(ctx, request.UID, "veryRandom123")
			ctx = context.WithValue(ctx, request.StartTime, time.Now())
			ctx = context.WithValue(ctx, request.APIName, "myapi")
			//ctx = context.WithValue(ctx, request.ServerName, "myTestServer")
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			rw := writer.NewResponseWriter(rec, "")

			handler.ServeHTTP(rw, req)
			accessLog.Do(rw, req)

			entry := hook.LastEntry()
			for key, expFields := range tc.expFields {
				value, exist := entry.Data[key]
				if !exist {
					subT.Errorf("Expected log field %s, got nothing", key)
				}

				switch ef := expFields.(type) {
				case logrus.Fields:
					for k, expected := range ef {
						var result interface{}
						switch fields := value.(type) {
						case logging.Fields:
							r, ok := fields[k]
							if !ok {
								subT.Errorf("Expected log field %s.%s, got nothing", key, k)
							}
							result = r
						}

						if !reflect.DeepEqual(expected, result) {
							subT.Errorf("Want: %v for key %s, got: %v", expected, k, result)
						}
					}
				default:
					if !reflect.DeepEqual(expFields, value) {
						subT.Errorf("Want: %v for key %s, got: %v", expFields, key, value)
					}
				}
			}
		})
	}
}
