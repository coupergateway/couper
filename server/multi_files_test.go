package server_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestMultiFiles_Server(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, hook := newCouperMultiFiles(
		"testdata/multi/server/couper.hcl",
		"testdata/multi/server/couper.d",
		helper,
	)
	defer shutdown()

	for _, entry := range hook.AllEntries() {
		t.Log(entry.String())
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	if err == nil || err.Error() != `Get "http://example.com:8080/": dial tcp4 127.0.0.1:8080: connect: connection refused` {
		t.Error("Expected hosts port override to 9080")
	}

	type testcase struct {
		url       string
		expStatus int
		expBody   string
	}

	for _, tc := range []testcase{
		{"http://example.com:9080/", http.StatusOK, "<body>RIGHT INCLUDE</body>\n"},
		{"http://example.com:9080/free", http.StatusForbidden, ""},
		{"http://example.com:9080/api-111", http.StatusUnauthorized, ""},
		{"http://example.com:9080/api-3", http.StatusTeapot, ""},
		{"http://example.com:9080/api-4/ep", http.StatusNoContent, ""},
		{"http://example.com:9081/", http.StatusOK, ""},
		{"http://example.com:8082/", http.StatusOK, ""},
		{"http://example.com:8083/", http.StatusNotFound, ""},
		{"http://example.com:9084/", http.StatusNotFound, ""},
	} {
		req, err = http.NewRequest(http.MethodGet, tc.url, nil)
		helper.Must(err)

		res, err = client.Do(req)
		helper.Must(err)

		if res.StatusCode != tc.expStatus {
			t.Errorf("request %q: want status %d, got %d", tc.url, tc.expStatus, res.StatusCode)
		}

		if tc.expBody == "" {
			continue
		}

		resBytes, err := io.ReadAll(res.Body)
		helper.Must(err)
		_ = res.Body.Close()

		if string(resBytes) != tc.expBody {
			t.Errorf("request %q unexpected content: %s", tc.url, resBytes)
		}
	}
}

func TestMultiFiles_SettingsAndDefaults(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouperMultiFiles(
		"testdata/multi/settings/couper.hcl",
		"testdata/multi/settings/couper.d",
		helper,
	)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if !strings.Contains(string(resBytes), `"Req-Id-Be-Hdr":["`) {
		t.Errorf("%s", resBytes)
	}

	if res.Header.Get("Req-Id-Cl-Hdr") == "" {
		t.Errorf("Missing 'Req-Id-Cl-Hdr' header")
	}

	// Call health route
	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/xyz", nil)
	helper.Must(err)

	res, err = client.Do(req)
	helper.Must(err)

	resBytes, err = io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if s := string(resBytes); s != "healthy" {
		t.Errorf("Unexpected body given: %s", s)
	}
}

func TestMultiFiles_Definitions(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouperMultiFiles(
		"testdata/multi/definitions/couper.hcl",
		"testdata/multi/definitions/couper.d",
		helper,
	)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if s := string(resBytes); s != "1234567890" {
		t.Errorf("Unexpected body given: %s", s)
	}

	// Call protected route
	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/added", nil)
	helper.Must(err)

	res, err = client.Do(req)
	helper.Must(err)

	resBytes, err = io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if s := string(resBytes); !strings.Contains(s, "401 access control error") {
		t.Errorf("Unexpected body given: %s", s)
	}
}
