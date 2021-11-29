package server_test

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
)

func TestCustomLogs_Upstream(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/logs/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path  string
		expAL logrus.Fields
		expUL logrus.Fields
	}

	for _, tc := range []testCase{
		{
			"/",
			logrus.Fields{
				"api":      "couper test-backend",
				"endpoint": "couper test-backend",
				"server":   "couper test-backend",
			},
			logrus.Fields{
				"array": []interface{}{
					float64(1),
					"couper test-backend",
					[]interface{}{
						float64(2),
						"couper test-backend",
					},
					logrus.Fields{"x": "X"},
				},
				"bool":   true,
				"float":  1.23,
				"int":    float64(123),
				"object": logrus.Fields{"a": "A", "b": "B", "c": float64(123)},
				"req":    "GET",
				"string": "couper test-backend",
			},
		},
		{
			"/backend",
			logrus.Fields{"api": "couper test-backend", "server": "couper test-backend"},
			logrus.Fields{"backend": string("couper test-backend")},
		},
	} {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
		helper.Must(err)

		hook.Reset()
		_, err = client.Do(req)
		helper.Must(err)

		// Wait for logs
		time.Sleep(200 * time.Millisecond)

		// Access log
		got, ok := hook.AllEntries()[1].Data["custom"].(logrus.Fields)
		if !ok {
			t.Fatalf("expected\n%#v\ngot\n%#v", tc.expAL, got)
		}
		if !reflect.DeepEqual(tc.expAL, got) {
			t.Errorf("expected\n%#v\ngot\n%#v", tc.expAL, got)
		}

		// Upstream log
		got, ok = hook.AllEntries()[0].Data["custom"].(logrus.Fields)
		if !ok {
			t.Fatalf("expected\n%#v\ngot\n%#v", tc.expUL, got)
		}
		if !cmp.Equal(tc.expUL, got) {
			t.Fail()

			fmt.Println(cmp.Diff(tc.expUL, got))
		}
	}
}

func TestCustomLogs_Local(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/logs/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path string
		exp  logrus.Fields
	}

	for _, tc := range []testCase{
		{"/secure", logrus.Fields{"error_handler": "GET"}},
		{"/file.html", logrus.Fields{"files": "GET"}},
		{"/spa", logrus.Fields{"spa": "GET"}},
	} {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
		helper.Must(err)

		hook.Reset()
		_, err = client.Do(req)
		helper.Must(err)

		// Wait for logs
		time.Sleep(200 * time.Millisecond)

		// Access log
		entries := hook.AllEntries()
		if len(entries) == 0 {
			t.Errorf("expected log entries, got none")
			return
		}

		got, ok := entries[0].Data["custom"].(logrus.Fields)
		if !ok {
			t.Fatal("expected custom log field, got none")
		}

		if !reflect.DeepEqual(tc.exp, got) {
			t.Error(cmp.Diff(tc.exp, got))
		}
	}
}

func TestCustomLogs_Merge(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/logs/02_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	// Wait for logs
	time.Sleep(200 * time.Millisecond)

	exp := logrus.Fields{
		"api":      true,
		"endpoint": true,
		"l1":       "endpoint",
		"l2":       []interface{}{"server", "api", "endpoint"},
		"l3":       []interface{}{"endpoint"},
		"server":   true,
	}

	// Access log
	got, ok := hook.AllEntries()[0].Data["custom"].(logrus.Fields)
	if !ok {
		t.Fatalf("expected\n%#v\ngot\n%#v", exp, got)
	}
	if !reflect.DeepEqual(exp, got) {
		t.Errorf("expected\n%#v\ngot\n%#v", exp, got)
	}
}
