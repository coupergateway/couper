package logging_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/logging"
)

var _ http.RoundTripper = &testRoundTripper{}

type testRoundTripper struct {
	response *http.Response
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := *t.response
	resp.Request = req
	return &resp, nil
}

func TestUpstreamLog_RoundTrip(t *testing.T) {
	helper := test.New(t)
	logger, hook := test.NewLogger()

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
					"proto": "https",
				},
			},
		},
		{
			description: "proto http",
			url:         "http://example.com",
			expFields: logrus.Fields{
				"request": logrus.Fields{
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
				},
				"response": logrus.Fields{
					"status": 200,
				},
				"method": http.MethodGet,
				"status": 200,
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
					"port":   "8080",
				},
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
			},
		},
		{
			description: "required request fields",
			url:         "http://localhost:8080/test",
			expFields: logrus.Fields{
				"request": logrus.Fields{
					"host":   "localhost",
					"origin": "localhost:8080",
					"path":   "/test",
					"method": http.MethodGet,
					"proto":  "http",
					"port":   "8080",
				},
				"response": logrus.Fields{
					"status": 200,
				},
				"method": http.MethodGet,
				"uid":    nil,
				"status": 200,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(subT *testing.T) {

			hook.Reset()

			myRT := &testRoundTripper{
				response: &http.Response{
					StatusCode: http.StatusOK,
				},
			}

			upstreamLog := logging.NewUpstreamLog(logger.WithContext(context.TODO()), myRT, true)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			_, err := upstreamLog.RoundTrip(req)
			helper.Must(err)

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
