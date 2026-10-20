package server_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/logging"
)

// pathCase describes one client request against a path normalization fixture.
type pathCase struct {
	path string
	// status is the expected response status.
	status int
	// body is a substring the response body must contain. Empty skips the check.
	body string
	// origin reports whether the gateway may call the origin backend.
	origin bool
	// logPath is the expected access log request path. Empty skips the check.
	logPath string
	// rawPath is the expected access log raw path. Empty requires the field to be absent.
	rawPath string
	// varPath is the expected request.path variable, logged as custom field. Empty skips the check.
	varPath string
}

// newRawBackend serves the protected resource and echoes every other request
// target byte for byte. It uses no ServeMux, so nothing cleans the path.
func newRawBackend(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/admin/panel.txt" {
			_, _ = rw.Write([]byte("TOP SECRET admin panel"))
			return
		}
		_, _ = fmt.Fprintf(rw, "origin saw %s", req.RequestURI)
	}))
	t.Cleanup(srv.Close)

	if err := os.Setenv("COUPER_TEST_RAW_BACKEND_ADDR", srv.URL); err != nil {
		t.Fatal(err)
	}

	return srv
}

func runPathCases(t *testing.T, fixture, base string, cases []pathCase) {
	t.Helper()

	newRawBackend(t)
	helper := test.New(t)
	shutdown, hook := newCouper(fixture, helper)
	defer shutdown()

	client := newClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	for _, tc := range cases {
		t.Run(tc.path, func(subT *testing.T) {
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, base+tc.path, nil)
			if err != nil {
				subT.Fatal(err)
			}
			// http.NewRequest keeps dot segments and encodings as given.
			if got := req.URL.RequestURI(); got != tc.path {
				subT.Fatalf("request target changed by the client: %q", got)
			}

			res, err := client.Do(req)
			if err != nil {
				subT.Fatal(err)
			}
			body, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()

			if res.StatusCode != tc.status {
				subT.Errorf("status: want %d, got %d, body: %q", tc.status, res.StatusCode, string(body))
			}
			if tc.body != "" && !strings.Contains(string(body), tc.body) {
				subT.Errorf("body: want substring %q, got %q", tc.body, string(body))
			}

			access, backend := waitForAccessLog(hook)
			if access == nil {
				subT.Fatal("missing access log entry")
			}
			if backend != tc.origin {
				subT.Errorf("origin call: want %v, got %v", tc.origin, backend)
			}

			reqFields, _ := access.Data["request"].(logging.Fields)
			if tc.logPath != "" {
				if p, _ := reqFields["path"].(string); p != tc.logPath {
					subT.Errorf("access log path: want %q, got %q", tc.logPath, p)
				}
			}
			if raw, ok := reqFields["raw_path"]; ok != (tc.rawPath != "") || (ok && raw != tc.rawPath) {
				subT.Errorf("access log raw_path: want %q, got %q", tc.rawPath, raw)
			}
			if tc.varPath != "" {
				custom, _ := access.Data["custom"].(logrus.Fields)
				if p, _ := custom["path"].(string); p != tc.varPath {
					subT.Errorf("request.path: want %q, got %q", tc.varPath, p)
				}
			}
		})
	}
}

// waitForAccessLog returns the client access log entry of the last request
// and whether a backend request was logged for it.
func waitForAccessLog(hook *logrustest.Hook) (*logrus.Entry, bool) {
	var access *logrus.Entry
	backend := false
	for i := 0; i < 100; i++ {
		for _, entry := range hook.AllEntries() {
			switch entry.Data["type"] {
			case "couper_access":
				access = entry
			case "couper_backend":
				backend = true
			}
		}
		if access != nil {
			return access, backend
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, backend
}

func TestPathNormalization_AccessControl(t *testing.T) {
	runPathCases(t, "testdata/integration/path_normalization/01_couper.hcl", "http://example.com:8080", []pathCase{
		{path: "/admin/panel.txt", status: 401, logPath: "/admin/panel.txt"},
		{path: "//admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "//admin/panel.txt"},
		{path: "/./admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/./admin/panel.txt"},
		{path: "/x/../admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/../admin/panel.txt", varPath: "/admin/panel.txt"},
		{path: "/x/%2e%2e/admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/%2e%2e/admin/panel.txt", varPath: "/admin/panel.txt"},
		{path: "/x/.%2E/admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/.%2E/admin/panel.txt"},
		{path: "/%61dmin/panel.txt", status: 401, logPath: "/%61dmin/panel.txt"},
		{path: "/admin/.", status: 401, logPath: "/admin/", rawPath: "/admin/."},
		{path: "/admin/..", status: 200, body: "origin saw /", origin: true, logPath: "/", rawPath: "/admin/.."},
		{path: "/../admin/panel.txt", status: 400, logPath: "/../admin/panel.txt"},
		{path: "/%2e%2e/admin/panel.txt", status: 400, logPath: "/%2e%2e/admin/panel.txt"},
		{path: "/x%2F..%2Fadmin/panel.txt", status: 400, logPath: "/x%2F..%2Fadmin/panel.txt"},
		{path: "/admin%2F..", status: 400, logPath: "/admin%2F.."},
		{path: "/a%2F%2Fb", status: 400, logPath: "/a%2F%2Fb"},
	})
}

func TestPathNormalization_Upstream(t *testing.T) {
	runPathCases(t, "testdata/integration/path_normalization/01_couper.hcl", "http://example.com:8080", []pathCase{
		{path: "/x//y", status: 200, body: "origin saw /x/y", origin: true, logPath: "/x/y", rawPath: "/x//y", varPath: "/x/y"},
		{path: "/x/z/../y", status: 200, body: "origin saw /x/y", origin: true, logPath: "/x/y", rawPath: "/x/z/../y", varPath: "/x/y"},
		{path: "/a%2Fb", status: 200, body: "origin saw /a%2Fb", origin: true, logPath: "/a%2Fb", varPath: "/a/b"},
		{path: "/%78/y", status: 200, body: "origin saw /%78/y", origin: true, logPath: "/%78/y"},
		{path: "/x/y/", status: 200, body: "origin saw /x/y/", origin: true, logPath: "/x/y/"},
		{path: "/x/y?q=../", status: 200, body: "origin saw /x/y?q=../", origin: true, logPath: "/x/y?q=../"},
	})
}

func TestPathNormalization_RateLimiter(t *testing.T) {
	runPathCases(t, "testdata/integration/path_normalization/01_couper.hcl", "http://example.com:8080", []pathCase{
		{path: "/limited", status: 200, body: "limited", logPath: "/limited"},
		{path: "//limited", status: 429, logPath: "/limited", rawPath: "//limited"},
		{path: "/x/../limited", status: 429, logPath: "/limited", rawPath: "/x/../limited"},
		// The trailing slash is not a normalization case, so this is a second key.
		{path: "/limited/", status: 200, body: "limited", logPath: "/limited/"},
	})
}

func TestPathNormalization_Rewrites(t *testing.T) {
	t.Run("path", func(t *testing.T) {
		runPathCases(t, "testdata/integration/path_normalization/02_couper.hcl", "http://example.com:8080", []pathCase{
			{path: "/admin/panel.txt", status: 401},
			{path: "/x/../admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/../admin/panel.txt"},
			{path: "/x/%2e%2e/admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/%2e%2e/admin/panel.txt"},
			{path: "/x%2F..%2Fadmin/panel.txt", status: 400},
			{path: "/x/../../admin/panel.txt", status: 400},
			{path: "/x/y/../z", status: 200, body: "origin saw /files/z", origin: true, logPath: "/x/z", rawPath: "/x/y/../z"},
			{path: "/x/z", status: 200, body: "origin saw /files/z", origin: true, logPath: "/x/z"},
		})
	})

	t.Run("path_prefix", func(t *testing.T) {
		runPathCases(t, "testdata/integration/path_normalization/02_couper.hcl", "http://example.com:9897", []pathCase{
			{path: "/admin/panel.txt", status: 401},
			{path: "/x/../admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/../admin/panel.txt"},
			{path: "/x/../../admin/panel.txt", status: 400},
			{path: "/x/%2e%2e/%2e%2e/admin/panel.txt", status: 400},
			{path: "/x/y/../z", status: 200, body: "origin saw /files/x/z", origin: true, logPath: "/x/z", rawPath: "/x/y/../z"},
		})
	})
}

func TestPathNormalization_FixedSegments(t *testing.T) {
	runPathCases(t, "testdata/integration/path_normalization/03_couper.hcl", "http://example.com:8080", []pathCase{
		{path: "/admin/panel.txt", status: 401},
		{path: "/x/../admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/x/../admin/panel.txt"},
		{path: "/./admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "/./admin/panel.txt"},
		{path: "//admin/panel.txt", status: 401, logPath: "/admin/panel.txt", rawPath: "//admin/panel.txt"},
		{path: "/x/panel.txt", status: 200, body: "origin saw /x/panel.txt", origin: true},
		{path: "/admin/..", status: 404, logPath: "/", rawPath: "/admin/.."},
	})
}

// rawRequest sends the request target byte for byte; net/http clients would
// re-encode characters such as "|".
func rawRequest(t *testing.T, addr, target string) int {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n", target)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)

	return res.StatusCode
}

func TestPathNormalization_RawPathBytes(t *testing.T) {
	newRawBackend(t)
	helper := test.New(t)
	shutdown, hook := newCouper("testdata/integration/path_normalization/01_couper.hcl", helper)
	defer shutdown()

	hook.Reset()
	if status := rawRequest(t, "127.0.0.1:8080", "/a|b/../admin/panel.txt"); status != http.StatusUnauthorized {
		t.Errorf("status: want 401, got %d", status)
	}

	access, _ := waitForAccessLog(hook)
	if access == nil {
		t.Fatal("missing access log entry")
	}
	reqFields, _ := access.Data["request"].(logging.Fields)
	if p, _ := reqFields["path"].(string); p != "/admin/panel.txt" {
		t.Errorf("access log path: want %q, got %q", "/admin/panel.txt", p)
	}
	if raw, _ := reqFields["raw_path"].(string); raw != "/a|b/../admin/panel.txt" {
		t.Errorf("access log raw_path: want %q, got %q", "/a|b/../admin/panel.txt", raw)
	}
}

func TestPathNormalization_Files(t *testing.T) {
	runPathCases(t, "testdata/integration/path_normalization/04_couper.hcl", "http://example.com:8080", []pathCase{
		{path: "/private/secret.txt", status: 401},
		{path: "//private/secret.txt", status: 401, logPath: "/private/secret.txt", rawPath: "//private/secret.txt"},
		{path: "/private/../private/secret.txt", status: 401, logPath: "/private/secret.txt", rawPath: "/private/../private/secret.txt"},
		{path: "/sub/../index.html", status: 200, body: "public index", logPath: "/index.html", rawPath: "/sub/../index.html"},
		{path: "/index.html", status: 200, body: "public index", logPath: "/index.html"},
	})
}

func TestPathNormalization_URLAttribute(t *testing.T) {
	runPathCases(t, "testdata/integration/path_normalization/05_couper.hcl", "http://example.com:8080", []pathCase{
		{path: "/u?f=x", status: 200, body: "origin saw /files/x", origin: true},
		{path: "/u?f=../admin/panel.txt", status: 500},
		{path: "/u?f=%2e%2e/admin/panel.txt", status: 500},
		{path: "/u?f=%252e%252e/admin/panel.txt", status: 500},
		{path: "/u?f=x%2F..%2Fadmin/panel.txt", status: 500},
		{path: "/r?f=x", status: 200, body: "origin saw /files/x", origin: true},
		{path: "/r?f=../admin/panel.txt", status: 500},
	})
}
