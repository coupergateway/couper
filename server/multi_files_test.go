package server_test

import (
	"io"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avenga/couper/config/configload"
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

	_, err = client.Do(req)
	if err == nil || err.Error() != `Get "http://example.com:8080/": dial tcp4 127.0.0.1:8080: connect: connection refused` {
		t.Error("Expected hosts port override to 9080")
	}

	type testcase struct {
		url       string
		header    test.Header
		expStatus int
		expBody   string
	}

	token := "eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.AZ0gZVqPe9TjjjJO0GnlTvERBXhPyxW_gTn050rCoEkseFRlp4TYry7WTQ7J4HNrH3btfxaEQLtTv7KooVLXQyMDujQbKU6cyuYH6MZXaM0Co3Bhu0awoX-2GVk997-7kMZx2yvwIR5ypd1CERIbNs5QcQaI4sqx_8oGrjO5ZmOWRqSpi4Mb8gJEVVccxurPu65gPFq9esVWwTf4cMQ3GGzijatnGDbRWs_igVGf8IAfmiROSVd17fShQtfthOFd19TGUswVAleOftC7-DDeJgAK8Un5xOHGRjv3ypK_6ZLRonhswaGXxovE0kLq4ZSzumQY2hOFE6x_BbrR1WKtGw"

	for _, tc := range []testcase{
		{"http://example.com:9080/", nil, http.StatusOK, "<body>RIGHT INCLUDE</body>\n"},
		{"http://example.com:9080/free", nil, http.StatusForbidden, ""},
		{"http://example.com:9080/errors/", test.Header{"Authorization": "Bearer " + token}, http.StatusTeapot, ""},
		{"http://example.com:9080/api-111", nil, http.StatusUnauthorized, ""},
		{"http://example.com:9080/api-3", nil, http.StatusTeapot, ""},
		{"http://example.com:9080/api-4/ep", nil, http.StatusNoContent, ""},
		{"http://example.com:9081/", nil, http.StatusOK, ""},
		{"http://example.com:8082/", nil, http.StatusOK, ""},
		{"http://example.com:8083/", nil, http.StatusNotFound, ""},
		{"http://example.com:9084/", nil, http.StatusNotFound, ""},
	} {
		req, err = http.NewRequest(http.MethodGet, tc.url, nil)
		helper.Must(err)

		for k, v := range tc.header {
			req.Header.Set(k, v)
		}

		res, err := client.Do(req)
		helper.Must(err)

		if res.StatusCode != tc.expStatus {
			t.Errorf("request %q: want status %d, got %d", tc.url, tc.expStatus, res.StatusCode)
		}

		if tc.expBody == "" {
			continue
		}

		var resBytes []byte
		resBytes, err = io.ReadAll(res.Body)
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

	if res.Header.Get("X") != "X" {
		t.Errorf("Invalid 'X' header given")
	}

	if res.Header.Get("Y") != "Y" {
		t.Errorf("Invalid 'Y' header given")
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

func TestMultiFiles_MultipleBackends(t *testing.T) {
	type testCase struct {
		config string
		blType string
	}

	for _, tc := range []testCase{
		{"testdata/multi/backends/errors/jwt.hcl", "jwt"},
		{"testdata/multi/backends/errors/beta_oauth2.hcl", "beta_oauth2"},
		{"testdata/multi/backends/errors/oidc.hcl", "oidc"},
		{"testdata/multi/backends/errors/ac_eh.hcl", "proxy"},
		{"testdata/multi/backends/errors/ep_proxy.hcl", "proxy"},
		{"testdata/multi/backends/errors/ep_request.hcl", "request"},
		{"testdata/multi/backends/errors/api_ep.hcl", "proxy"},
	} {
		t.Run(tc.config, func(st *testing.T) {
			_, err := configload.LoadFile(filepath.Join(testWorkingDir, tc.config), "")

			if !strings.HasSuffix(err.Error(), fmt.Sprintf(": Multiple definitions of backend are not allowed in %s.; ", tc.blType)) {
				st.Errorf("Unexpected error: %s", err.Error())
			}
		})
	}
}

func Test_MultipleLabels(t *testing.T) {
	type testCase struct {
		name       string
		configPath string
		expError   string
	}

	for _, tc := range []testCase{
		{
			"server with multiple labels",
			"testdata/multi/errors/couper_01.hcl",
			"testdata/multi/errors/couper_01.hcl:1,12-15: Extraneous label for server; Only 1 labels (name) are expected for server blocks.",
		},
		{
			"api with multiple labels",
			"testdata/multi/errors/couper_02.hcl",
			"testdata/multi/errors/couper_02.hcl:2,11-14: Extraneous label for api; Only 1 labels (name) are expected for api blocks.",
		},
		{
			"spa with multiple labels",
			"testdata/multi/errors/couper_04.hcl",
			"testdata/multi/errors/couper_04.hcl:2,11-14: Extraneous label for spa; Only 1 labels (name) are expected for spa blocks.",
		},
		{
			"files with multiple labels",
			"testdata/multi/errors/couper_05.hcl",
			"testdata/multi/errors/couper_05.hcl:2,13-16: Extraneous label for files; Only 1 labels (name) are expected for files blocks.",
		},
		{
			"api, spa, files and server without labels",
			"testdata/multi/errors/couper_03.hcl",
			"",
		},
	} {
		t.Run(tc.name, func(st *testing.T) {
			_, err := configload.LoadFile(filepath.Join(testWorkingDir, tc.configPath), "")

			if (err != nil && tc.expError == "") ||
				(tc.expError != "" && (err == nil || !strings.Contains(err.Error(), tc.expError))) {
				st.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMultiFiles_SPAs(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouperMultiFiles("", "testdata/multi/server/spa.d", helper)
	defer shutdown()

	type testcase struct {
		path        string
		expStatus   int
		expContains string
	}

	for _, tc := range []testcase{
		{"/", http.StatusNotFound, ""},
		{"/app", http.StatusOK, "02_spa.hcl"},
		{"/another", http.StatusOK, "03_spa.hcl"},
	} {
		t.Run(tc.path, func(st *testing.T) {
			h := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://couper.local:8080"+tc.path, nil)
			h.Must(err)

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Errorf("want status: %d, got: %d", tc.expStatus, res.StatusCode)
			}

			b, err := io.ReadAll(res.Body)
			h.Must(err)

			h.Must(res.Body.Close())

			if tc.expContains != "" && !strings.Contains(string(b), tc.expContains) {
				st.Errorf("want %q, got:\n%q", tc.expContains, string(b))
			}
		})
	}
}

func TestMultiFiles_Files(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouperMultiFiles("", "testdata/multi/server/files.d", helper)
	defer shutdown()

	type testcase struct {
		path        string
		expStatus   int
		expContains string
	}

	for _, tc := range []testcase{
		{"/", http.StatusNotFound, ""},
		{"/app", http.StatusOK, "<app/>\n"},
		{"/another", http.StatusOK, "<another/>\n"},
	} {
		t.Run(tc.path, func(st *testing.T) {
			h := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://couper.local:8080"+tc.path, nil)
			h.Must(err)

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Errorf("want status: %d, got: %d", tc.expStatus, res.StatusCode)
			}

			b, err := io.ReadAll(res.Body)
			h.Must(err)

			h.Must(res.Body.Close())

			if tc.expContains != "" && !strings.Contains(string(b), tc.expContains) {
				st.Errorf("want %q, got:\n%q", tc.expContains, string(b))
			}
		})
	}
}
