package server_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

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
