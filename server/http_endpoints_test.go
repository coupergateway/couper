package server_test

import (
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/avenga/couper/internal/test"
)

const testdataPath = "testdata/endpoints"

func TestEndpoints_ProxyReqRes(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "01_couper.hcl"), test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 3 {
		t.Fatalf("Expected 3 log entries, given %d", l)
	}
	t.Errorf("%#v", res.Status)
	t.Errorf("%#v", entries[0].Data)
	t.Errorf("%#v", entries[1].Data)
	t.Errorf("%#v", entries[2].Data)

	resBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	helper.Must(err)

	t.Errorf("%s", resBytes)
}

func TestEndpoints_Res(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "02_couper.hcl"), test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 1 {
		t.Fatalf("Expected 1 log entries, given %d", l)
	}
	t.Errorf("%#v", res.Status)
	t.Errorf("%#v", entries[0].Data)

	resBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	helper.Must(err)

	t.Errorf("%s", resBytes)
}
