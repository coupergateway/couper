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

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	// Wait for logs
	time.Sleep(200 * time.Millisecond)

	// Access log
	exp := logrus.Fields{"api": "GET", "endpoint": "GET", "server": "GET"}
	got, ok := hook.AllEntries()[1].Data["custom"].(logrus.Fields)
	if !ok {
		t.Fatalf("expected\n%#v\ngot\n%#v", exp, got)
	}
	if !reflect.DeepEqual(exp, got) {
		t.Errorf("expected\n%#v\ngot\n%#v", exp, got)
	}

	// Upstream log
	exp = logrus.Fields{
		"array":  []interface{}{float64(1), "GET", []interface{}{float64(2), "GET"}, logrus.Fields{"x": "X"}},
		"bool":   true,
		"float":  1.23,
		"int":    float64(123),
		"object": logrus.Fields{"a": "A", "b": "B", "c": float64(123)},
		"string": "GET",
	}
	got, ok = hook.AllEntries()[0].Data["custom"].(logrus.Fields)
	if !ok {
		t.Fatalf("expected\n%#v\ngot\n%#v", exp, got)
	}
	if !cmp.Equal(exp, got) {
		t.Fail()

		fmt.Println(cmp.Diff(exp, got))
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
		got, ok := hook.AllEntries()[0].Data["custom"].(logrus.Fields)
		if !ok {
			t.Fatalf("expected\n%#v\ngot\n%#v", tc.exp, got)
		}
		if !reflect.DeepEqual(tc.exp, got) {
			t.Errorf("expected\n%#v\ngot\n%#v", tc.exp, got)
		}
	}
}
