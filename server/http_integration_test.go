package server_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/oauth2"
)

var (
	testBackend    *test.Backend
	testWorkingDir string
	testProxyAddr  = "http://127.0.0.1:9999"
	testServerMu   = sync.Mutex{}
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	println("INTEGRATION: create test backend...")
	testBackend = test.NewBackend()
	err := os.Setenv("COUPER_TEST_BACKEND_ADDR", testBackend.Addr())
	if err != nil {
		panic(err)
	}

	err = os.Setenv("HTTP_PROXY", testProxyAddr)
	if err != nil {
		panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testWorkingDir = wd
}

func teardown() {
	println("INTEGRATION: close test backend...")
	for _, key := range []string{"COUPER_TEST_BACKEND_ADDR", "HTTP_PROXY"} {
		if err := os.Unsetenv(key); err != nil {
			panic(err)
		}
	}
	testBackend.Close()
}

func newCouper(file string, helper *test.Helper) (func(), *logrustest.Hook) {
	couperConfig, err := configload.LoadFile(filepath.Join(testWorkingDir, file), "test")
	helper.Must(err)

	return newCouperWithConfig(couperConfig, helper)
}

func newCouperMultiFiles(file, dir string, helper *test.Helper) (func(), *logrustest.Hook) {
	couperConfig, err := configload.LoadFiles([]string{file, dir}, "test")
	helper.Must(err)

	return newCouperWithConfig(couperConfig, helper)
}

// newCouperWithTemplate applies given variables first and loads Couper with the resulting configuration file.
// Example template:
//
//	My {{.message}}
//
// Example value:
//
//	map[string]interface{}{
//		"message": "value",
//	}
func newCouperWithTemplate(file string, helper *test.Helper, vars map[string]interface{}) (func(), *logrustest.Hook, error) {
	if vars == nil {
		s, h := newCouper(file, helper)
		return s, h, nil
	}

	tpl, err := template.New(filepath.Base(file)).ParseFiles(file)
	helper.Must(err)

	result := &bytes.Buffer{}
	helper.Must(tpl.Execute(result, vars))

	return newCouperWithBytes(result.Bytes(), helper)
}

func newCouperWithBytes(file []byte, helper *test.Helper) (func(), *logrustest.Hook, error) {
	couperConfig, err := configload.LoadBytes(file, "couper-bytes.hcl")
	if err != nil {
		return nil, nil, err
	}
	s, h := newCouperWithConfig(couperConfig, helper)
	return s, h, nil
}

func newCouperWithConfig(couperConfig *config.Couper, helper *test.Helper) (func(), *logrustest.Hook) {
	testServerMu.Lock()
	defer testServerMu.Unlock()

	log, hook := test.NewLogger()
	log.Level, _ = logrus.ParseLevel(couperConfig.Settings.LogLevel)

	ctx, cancelFn := context.WithCancel(context.Background())
	shutdownFn := func() {
		if helper.TestFailed() { // log on error
			time.Sleep(time.Second)
			for _, entry := range hook.AllEntries() {
				s, _ := entry.String()
				helper.Logf(s)
			}
		}
		cleanup(cancelFn, helper)
	}

	// ensure the previous test aren't listening
	port := couperConfig.Settings.DefaultPort
	test.WaitForClosedPort(port)
	waitForCh := make(chan struct{}, 1)
	command.RunCmdTestCallback = func() {
		waitForCh <- struct{}{}
	}
	defer func() { command.RunCmdTestCallback = nil }()

	go func() {
		if err := command.NewRun(ctx).Execute(nil, couperConfig, log.WithContext(ctx)); err != nil {
			command.RunCmdTestCallback()
			shutdownFn()
			if lerr, ok := err.(*errors.Error); ok {
				panic(lerr.LogError())
			} else {
				panic(err)
			}
		}
	}()
	<-waitForCh

	for _, entry := range hook.AllEntries() {
		if entry.Level < logrus.WarnLevel {
			// ignore health-check startup errors
			if req, ok := entry.Data["request"]; ok {
				if reqFields, ok := req.(logging.Fields); ok {
					n := reqFields["name"]
					if hc, ok := n.(string); ok && hc == "health-check" {
						continue
					}
				}
			}
			defer os.Exit(1) // ok in loop, next line is the end
			helper.Must(fmt.Errorf("error: %#v: %s", entry.Data, entry.Message))
		}
	}

	hook.Reset() // no startup logs
	return shutdownFn, hook
}

func newClient() *http.Client {
	return test.NewHTTPClient()
}

func cleanup(shutdown func(), helper *test.Helper) {
	testServerMu.Lock()
	defer testServerMu.Unlock()

	shutdown()

	err := os.Chdir(testWorkingDir)
	if err != nil {
		helper.Must(err)
	}
}

func TestHTTPServer_ServeHTTP(t *testing.T) {
	type testRequest struct {
		method, url string
	}

	type expectation struct {
		status      int
		body        []byte
		header      http.Header
		handlerName string
	}

	type requestCase struct {
		req testRequest
		exp expectation
	}

	type testCase struct {
		fileName string
		requests []requestCase
	}

	client := newClient()

	for i, testcase := range []testCase{
		{"spa/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusOK, []byte(`<html><body><title>1.0</title></body></html>`), nil, "spa"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/app"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
		}},
		{"files/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusOK, []byte(`<html lang="en">index</html>`), nil, "file"},
			},
		}},
		{"files/02_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/a"},
				expectation{http.StatusOK, []byte(`<html lang="en">index A</html>`), nil, "file"},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:9898/a"},
				expectation{http.StatusOK, []byte(`<html lang="en">index A</html>`), nil, "file"},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:9898/"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/b"},
				expectation{http.StatusOK, []byte(`<html lang="en">index B</html>`), nil, "file"},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
		}},
		{"files_spa_api/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusOK, []byte("<html><body><title>SPA_01</title>{\"default\":\"true\"}</body></html>\n"), nil, "spa"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/foo"},
				expectation{http.StatusOK, []byte("<html><body><title>SPA_01</title>{\"default\":\"true\"}</body></html>\n"), nil, "spa"},
			},
		}},
		{"api/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1"},
				expectation{http.StatusOK, nil, http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1/"},
				expectation{http.StatusOK, nil, http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1/not-found"},
				expectation{http.StatusNotFound, []byte(`{"message": "route not found error" }` + "\n"), http.Header{"Content-Type": {"application/json"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1/connect-error/"}, // in this case proxyconnect fails
				expectation{http.StatusBadGateway, []byte(`{"message": "backend error" }` + "\n"), http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1x"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
		}},
		{"api/02_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v2/"},
				expectation{http.StatusOK, nil, http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:9898/v2/"},
				expectation{http.StatusOK, nil, http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/v3/"},
				expectation{http.StatusOK, nil, http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v2/not-found"},
				expectation{http.StatusNotFound, []byte(`{"message": "route not found error" }` + "\n"), http.Header{"Content-Type": {"application/json"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:9898/v2/not-found"},
				expectation{http.StatusNotFound, []byte(`{"message": "route not found error" }` + "\n"), http.Header{"Content-Type": {"application/json"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/v3/not-found"},
				expectation{http.StatusNotFound, []byte(`{"message": "route not found error" }` + "\n"), http.Header{"Content-Type": {"application/json"}}, ""},
			},
		}},
		{"vhosts/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/notfound"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusOK, []byte("<html><body><title>FS_01</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "file"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/spa1"},
				expectation{http.StatusOK, []byte("<html><body><title>SPA_01</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "spa"},
			},
			{
				testRequest{http.MethodGet, "http://example.com:8080/"},
				expectation{http.StatusOK, []byte("<html><body><title>FS_01</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "file"},
			},
			{
				testRequest{http.MethodGet, "http://example.org:9876/"},
				expectation{http.StatusOK, []byte("<html><body><title>FS_01</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "file"},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:8080/"},
				expectation{http.StatusOK, []byte("<html><body><title>FS_02</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "file"},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:8080/spa2"},
				expectation{http.StatusOK, []byte("<html><body><title>SPA_02</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "spa"},
			},
			{
				testRequest{http.MethodGet, "http://example.net:9876/"},
				expectation{http.StatusOK, []byte("<html><body><title>FS_02</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "file"},
			},
			{
				testRequest{http.MethodGet, "http://v-server3.com:8080/"},
				expectation{http.StatusOK, []byte("<html><body><title>FS_03</title></body></html>\n"), http.Header{"Content-Type": {"text/html; charset=utf-8"}}, "file"},
			},
			{
				testRequest{http.MethodGet, "http://v-server3.com:8080/spa2"},
				expectation{http.StatusNotFound, []byte("<html>route not found error</html>\n"), http.Header{"Couper-Error": {"route not found error"}}, ""},
			},
		}},
		{"endpoint_eval/16_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusInternalServerError, []byte("<html>configuration error</html>\n"), http.Header{"Couper-Error": {"configuration error"}}, ""},
			},
		}},
	} {
		confPath := path.Join("testdata/integration", testcase.fileName)
		t.Logf("#%.2d: Create Couper: %q", i+1, confPath)

		for _, rc := range testcase.requests {
			t.Run(testcase.fileName+" "+rc.req.method+"|"+rc.req.url, func(subT *testing.T) {
				helper := test.New(subT)
				shutdown, logHook := newCouper(confPath, helper)
				defer shutdown()

				logHook.Reset()

				req, err := http.NewRequest(rc.req.method, rc.req.url, nil)
				helper.Must(err)

				res, err := client.Do(req)
				helper.Must(err)

				resBytes, err := io.ReadAll(res.Body)
				helper.Must(err)

				_ = res.Body.Close()

				if res.StatusCode != rc.exp.status {
					subT.Errorf("Expected statusCode %d, got %d", rc.exp.status, res.StatusCode)
					subT.Logf("Failed: %s|%s", testcase.fileName, rc.req.url)
				}

				for k, v := range rc.exp.header {
					if !reflect.DeepEqual(res.Header[k], v) {
						subT.Errorf("Exptected headers:\nWant:\t%#v\nGot:\t%#v\n", v, res.Header[k])
					}
				}

				if rc.exp.body != nil && !bytes.Equal(resBytes, rc.exp.body) {
					subT.Errorf("Expected same body content:\nWant:\t%q\nGot:\t%q\n", string(rc.exp.body), string(resBytes))
				}

				entry := logHook.LastEntry()

				if entry == nil || entry.Data["type"] != "couper_access" {
					subT.Error("Expected a log entry, got nothing")
					return
				}
				if handler, ok := entry.Data["handler"]; rc.exp.handlerName != "" && (!ok || handler != rc.exp.handlerName) {
					subT.Errorf("Expected handler %q within logs, got:\n%#v", rc.exp.handlerName, entry.Data)
				}
			})
		}
	}
}

func TestHTTPServer_HostHeader(t *testing.T) {
	helper := test.New(t)

	client := newClient()

	confPath := path.Join("testdata/integration", "files/02_couper.hcl")
	shutdown, _ := newCouper(confPath, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:9898/b", nil)
	helper.Must(err)

	req.Host = "Example.com."
	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	if string(resBytes) != `<html lang="en">index B</html>` {
		t.Errorf("%s", resBytes)
	}
}

func TestHTTPServer_HostHeader2(t *testing.T) {
	helper := test.New(t)

	client := newClient()

	confPath := path.Join("testdata/integration", "api/03_couper.hcl")
	shutdown, logHook := newCouper(confPath, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://couper.io:9898/v3/def", nil)
	helper.Must(err)

	req.Host = "couper.io"
	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	if string(resBytes) != "<html>route not found error</html>\n" {
		t.Errorf("%s", resBytes)
	}

	entry := logHook.LastEntry()
	if entry == nil {
		t.Error("Expected a log entry, got nothing")
	} else if entry.Data["server"] != "multi-api-host1" {
		t.Errorf("Expected 'multi-api-host1', got: %s", entry.Data["server"])
	}
}

func TestHTTPServer_EnvVars(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	env.SetTestOsEnviron(func() []string {
		return []string{"BAP1=pass1"}
	})
	defer env.SetTestOsEnviron(os.Environ)

	shutdown, hook := newCouper("testdata/integration/env/01_couper.hcl", test.New(t))
	defer shutdown()

	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

func TestHTTPServer_DefaultEnvVars(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	env.SetTestOsEnviron(func() []string {
		return []string{"VALUE_4=value4"}
	})
	defer env.SetTestOsEnviron(os.Environ)

	shutdown, hook := newCouper("testdata/integration/env/02_couper.hcl", test.New(t))
	defer shutdown()

	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	var result []string
	helper.Must(json.Unmarshal(b, &result))

	if diff := cmp.Diff(result, []string{"value1", "", "default_value_3", "value4", "value5"}); diff != "" {
		t.Error(diff)
	}
}

func TestHTTPServer_XFHHeader(t *testing.T) {
	client := newClient()

	env.SetTestOsEnviron(func() []string {
		return []string{"COUPER_XFH=true"}
	})
	defer env.SetTestOsEnviron(os.Environ)

	confPath := path.Join("testdata/integration", "files/02_couper.hcl")
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()

	helper := test.New(t)
	logHook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:9898/b", nil)
	helper.Must(err)

	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Host", "example.com.")
	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	if string(resBytes) != `<html lang="en">index B</html>` {
		t.Errorf("%s", resBytes)
	}

	entry := logHook.LastEntry()
	if entry == nil {
		t.Error("Expected a log entry, got nothing")
	} else if entry.Data["server"] != "multi-files-host2" {
		t.Errorf("Expected 'multi-files-host2', got: %s", entry.Data["server"])
	} else if entry.Data["url"] != "http://example.com:9898/b" {
		t.Errorf("Expected 'http://example.com:9898/b', got: %s", entry.Data["url"])
	}
}

func TestHTTPServer_ProxyFromEnv(t *testing.T) {
	helper := test.New(t)

	seen := make(chan struct{})
	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
		go func() {
			seen <- struct{}{}
		}()
	}))
	ln, err := net.Listen("tcp4", testProxyAddr[7:])
	helper.Must(err)
	origin.Listener = ln
	origin.Start()
	defer func() {
		origin.Close()
		ln.Close()
		time.Sleep(time.Second)
	}()

	confPath := path.Join("testdata/integration", "api/01_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/v1/proxy", nil)
	helper.Must(err)

	_, err = newClient().Do(req)
	helper.Must(err)

	timer := time.NewTimer(time.Second)
	select {
	case <-timer.C:
		t.Error("Missing proxy call")
	case <-seen:
	}
}

func TestHTTPServer_Gzip(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration", "files/03_gzip.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	type testCase struct {
		name                 string
		headerAcceptEncoding string
		path                 string
		expectGzipResponse   bool
	}

	for _, tc := range []testCase{
		{"with mixed header AE gzip", "br, gzip", "/index.html", true},
		{"with header AE gzip", "gzip", "/index.html", true},
		{"with header AE and without gzip", "deflate", "/index.html", false},
		{"with header AE and space", " ", "/index.html", false},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.org:9898"+tc.path, nil)
			helper.Must(err)

			if tc.headerAcceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tc.headerAcceptEncoding)
			}

			res, err := client.Do(req)
			helper.Must(err)

			var body io.Reader
			body = res.Body

			if !tc.expectGzipResponse {
				if val := res.Header.Get("Content-Encoding"); val != "" {
					subT.Errorf("Expected no header with key Content-Encoding, got value: %s", val)
				}
			} else {
				if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
					subT.Errorf("Expected Content-Encoding header value: %q, got: %q", "gzip", ce)
				}

				body, err = gzip.NewReader(res.Body)
				helper.Must(err)
			}

			if vr := res.Header.Get("Vary"); vr != "Accept-Encoding" {
				subT.Errorf("Expected Accept-Encoding header value %q, got: %q", "Vary", vr)
			}

			resBytes, err := io.ReadAll(body)
			helper.Must(err)

			srcBytes, err := os.ReadFile(filepath.Join(testWorkingDir, "testdata/integration/files/htdocs_c_gzip"+tc.path))
			helper.Must(err)

			if !bytes.Equal(resBytes, srcBytes) {
				subT.Errorf("Want:\n%s\nGot:\n%s", string(srcBytes), string(resBytes))
			}
		})
	}
}

func TestHTTPServer_QueryParams(t *testing.T) {
	client := newClient()

	const confPath = "testdata/integration/endpoint_eval/"

	type expectation struct {
		Query url.Values
		Path  string
	}

	type testCase struct {
		file  string
		query string
		exp   expectation
	}

	for _, tc := range []testCase{
		{"04_couper.hcl", "a=b%20c&aeb_del=1&ae_del=1&CaseIns=1&caseIns=1&def_del=1&xyz=123", expectation{
			Query: url.Values{
				"a":           []string{"b c"},
				"ae_a_and_b":  []string{"A&B", "A&B"},
				"ae_empty":    []string{"", ""},
				"ae_multi":    []string{"str1", "str2", "str3", "str4"},
				"ae_string":   []string{"str", "str"},
				"ae":          []string{"ae", "ae"},
				"aeb_a_and_b": []string{"A&B", "A&B"},
				"aeb_empty":   []string{"", ""},
				"aeb_multi":   []string{"str1", "str2", "str3", "str4"},
				"aeb_string":  []string{"str", "str"},
				"aeb":         []string{"aeb", "aeb"},
				"caseIns":     []string{"1"},
				"def_del":     []string{"1"},
				"xxx":         []string{"aaa", "bbb"},
			},
			Path: "/",
		}},
		{"05_couper.hcl", "", expectation{
			Query: url.Values{
				"ae":  []string{"ae"},
				"def": []string{"def"},
			},
			Path: "/xxx",
		}},
		{"06_couper.hcl", "", expectation{
			Query: url.Values{
				"ae":  []string{"ae"},
				"def": []string{"def"},
			},
			Path: "/zzz",
		}},
		{"07_couper.hcl", "", expectation{
			Query: url.Values{
				"ae":  []string{"ae"},
				"def": []string{"def"},
			},
			Path: "/xxx",
		}},
		{"09_couper.hcl", "", expectation{
			Query: url.Values{
				"test": []string{"pest"},
			},
			Path: "/",
		}},
	} {
		t.Run("_"+tc.query, func(subT *testing.T) {
			helper := test.New(subT)

			shutdown, _ := newCouper(path.Join(confPath, tc.file), helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080?"+tc.query, nil)
			helper.Must(err)

			req.Header.Set("ae", "ae")
			req.Header.Set("aeb", "aeb")
			req.Header.Set("def", "def")
			req.Header.Set("xyz", "xyz")

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_PathPrefix(t *testing.T) {
	client := newClient()

	type expectation struct {
		Path string
	}

	type testCase struct {
		path string
		exp  expectation
	}

	for _, tc := range []testCase{
		{"/v1", expectation{
			Path: "/xxx/xxx/v1",
		}},
		{"/v1/vvv/foo", expectation{
			Path: "/xxx/xxx/api/foo",
		}},
		{"/v2/yyy", expectation{
			Path: "/v2/yyy",
		}},
		{"/v3/zzz", expectation{
			Path: "/zzz/v3/zzz",
		}},
	} {
		t.Run("_"+tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			shutdown, _ := newCouper("testdata/integration/api/06_couper.hcl", helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			// Test dynamic values in conf
			if strings.HasPrefix(tc.exp.Path, "/xxx") {
				req.Header.Set("X-Val", "xxx")
			}

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_BackendLogPath(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/07_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/?query#fragment", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	if p := hook.AllEntries()[0].Data["request"].(logging.Fields)["path"]; p != "/path?query" {
		t.Errorf("Unexpected path given: %s", p)
	}
}

func TestHTTPServer_BackendLogRequestProto(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/15_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	var backendLogsSeen int
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_access" {
			continue
		}

		backendLogsSeen++

		if p := entry.Data["request"].(logging.Fields)["proto"]; p != "http" {
			t.Errorf("want proto http, got: %q", p)
		}
	}

	if backendLogsSeen != 2 {
		t.Error("expected two backend request logs")
	}
}

func TestHTTPServer_PathInvalidFragment(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/09_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/?query#fragment", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	if m := hook.AllEntries()[0].Message; m != "configuration error: path attribute: invalid fragment found in \"/path#xxx\"" {
		t.Errorf("Unexpected message given: %s", m)
	}
}

func TestHTTPServer_PathInvalidQuery(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/10_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/?query#fragment", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	if m := hook.AllEntries()[0].Message; m != "configuration error: path attribute: invalid query string found in \"/path?xxx\"" {
		t.Errorf("Unexpected message given: %s", m)
	}
}

func TestHTTPServer_PathPrefixInvalidFragment(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/11_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/?query#fragment", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	if m := hook.AllEntries()[0].Message; m != "configuration error: path_prefix attribute: invalid fragment found in \"/path#xxx\"" {
		t.Errorf("Unexpected message given: %s", m)
	}
}

func TestHTTPServer_PathPrefixInvalidQuery(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/12_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/?query#fragment", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	if m := hook.AllEntries()[0].Message; m != "configuration error: path_prefix attribute: invalid query string found in \"/path?xxx\"" {
		t.Errorf("Unexpected message given: %s", m)
	}
}

func TestHTTPServer_RequestHeaders(t *testing.T) {
	client := newClient()

	const confPath = "testdata/integration/endpoint_eval/"

	type expectation struct {
		Headers http.Header
	}

	type testCase struct {
		file  string
		query string
		exp   expectation
	}

	for _, tc := range []testCase{
		{"12_couper.hcl", "ae=ae&aeb=aeb&def=def&xyz=xyz", expectation{
			Headers: http.Header{
				"Aeb":         []string{"aeb", "aeb"},
				"Aeb_a_and_b": []string{"A&B", "A&B"},
				"Aeb_empty":   []string{"", ""},
				"Aeb_multi":   []string{"str1", "str2", "str3", "str4"},
				"Aeb_string":  []string{"str", "str"},
				"Xxx":         []string{"aaa", "bbb"},
			},
		}},
	} {
		t.Run("_"+tc.query, func(subT *testing.T) {
			helper := test.New(subT)
			shutdown, _ := newCouper(path.Join(confPath, tc.file), helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080?"+tc.query, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if r1 := res.Header.Get("Remove-Me-1"); r1 != "r1" {
				subT.Errorf("Missing or invalid header Remove-Me-1: %s", r1)
			}
			if r2 := res.Header.Get("Remove-Me-2"); r2 != "" {
				subT.Errorf("Unexpected header %s", r2)
			}

			if s2 := res.Header.Get("Set-Me-2"); s2 != "s2" {
				subT.Errorf("Missing or invalid header Set-Me-2: %s", s2)
			}

			if a2 := res.Header.Get("Add-Me-2"); a2 != "a2" {
				subT.Errorf("Missing or invalid header Add-Me-2: %s", a2)
			}

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			jsonResult.Headers.Del("User-Agent")
			jsonResult.Headers.Del("X-Forwarded-For")
			jsonResult.Headers.Del("Couper-Request-Id")

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_LogFields(t *testing.T) {
	client := newClient()
	conf := "testdata/integration/endpoint_eval/10_couper.hcl"

	helper := test.New(t)
	shutdown, logHook := newCouper(conf, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.AllEntries()
	if l := len(entries); l != 2 {
		t.Fatalf("Unexpected number of log lines: %d", l)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	helper.Must(res.Body.Close())

	backendLog := entries[0]
	accessLog := entries[1]

	if tp, ok := backendLog.Data["type"]; !ok || tp != "couper_backend" {
		t.Fatalf("Unexpected log type: %s", tp)
	}
	if tp, ok := accessLog.Data["type"]; !ok || tp != "couper_access" {
		t.Fatalf("Unexpected log type: %s", tp)
	}

	if u, ok := backendLog.Data["url"]; !ok || u == "" {
		t.Fatalf("Unexpected URL: %s", u)
	}
	if u, ok := accessLog.Data["url"]; !ok || u == "" {
		t.Fatalf("Unexpected URL: %s", u)
	}

	if b, ok := backendLog.Data["backend"]; !ok || b != "anything" {
		t.Fatalf("Unexpected backend name: %s", b)
	}
	if e, ok := accessLog.Data["endpoint"]; !ok || e != "/" {
		t.Fatalf("Unexpected endpoint: %s", e)
	}

	if b, ok := accessLog.Data["response"].(logging.Fields)["bytes"]; !ok || b != len(resBytes) {
		t.Fatalf("Unexpected number of bytes: %d\npayload: %s", b, string(resBytes))
	}
}

func TestHTTPServer_QueryEncoding(t *testing.T) {
	client := newClient()

	conf := "testdata/integration/endpoint_eval/10_couper.hcl"

	type expectation struct {
		RawQuery string
	}

	helper := test.New(t)
	shutdown, _ := newCouper(conf, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080?a=a%20a&x=x+x", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	var jsonResult expectation
	err = json.Unmarshal(resBytes, &jsonResult)
	if err != nil {
		t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
	}

	exp := expectation{RawQuery: "a=a%20a&space=a%20b%2Bc&x=x%2Bx"}
	if !reflect.DeepEqual(jsonResult, exp) {
		t.Errorf("\nwant: \n%#v\ngot: \n%#v", exp, jsonResult)
	}
}

func TestHTTPServer_Backends(t *testing.T) {
	client := newClient()

	configPath := "testdata/integration/config/02_couper.hcl"

	helper := test.New(t)
	shutdown, _ := newCouper(configPath, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	exp := []string{"1", "4"}
	if !reflect.DeepEqual(res.Header.Values("Foo"), exp) {
		t.Errorf("\nwant: \n%#v\ngot: \n%#v", exp, res.Header.Values("Foo"))
	}
}

func TestHTTPServer_Backends_Reference(t *testing.T) {
	client := newClient()

	configPath := "testdata/integration/config/04_couper.hcl"

	helper := test.New(t)
	shutdown, _ := newCouper(configPath, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.Header.Get("proxy") != "a" || res.Header.Get("request") != "b" {
		t.Errorf("Expected proxy:a and request:b header values, got: %v", res.Header)
	}
}

func TestHTTPServer_Backends_Reference_BasicAuth(t *testing.T) {
	client := newClient()

	configPath := "testdata/integration/config/13_couper.hcl"

	helper := test.New(t)
	shutdown, _ := newCouper(configPath, helper)
	defer shutdown()

	type testcase struct {
		path     string
		wantAuth bool
	}

	for _, tc := range []testcase{
		{"/", false},
		{"/granted", true},
	} {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
		helper.Must(err)

		res, err := client.Do(req)
		helper.Must(err)

		b, err := io.ReadAll(res.Body)
		helper.Must(err)

		helper.Must(res.Body.Close())

		type result struct {
			Headers http.Header
		}
		r := result{}
		helper.Must(json.Unmarshal(b, &r))

		if tc.wantAuth && !strings.HasPrefix(r.Headers.Get("Authorization"), "Basic ") {
			t.Error("expected Authorization header value")
		}
	}
}

func TestHTTPServer_Backends_Reference_PathPrefix(t *testing.T) {
	client := newClient()

	configPath := "testdata/integration/config/12_couper.hcl"

	helper := test.New(t)
	shutdown, _ := newCouper(configPath, helper)
	defer shutdown()

	type testcase struct {
		path       string
		wantPath   string
		wantStatus int
	}

	for _, tc := range []testcase{
		{"/", "/anything", http.StatusOK},
		{"/prefixed", "/my-prefix/anything", http.StatusNotFound},
	} {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
		helper.Must(err)

		res, err := client.Do(req)
		helper.Must(err)

		type result struct {
			Path string
		}

		b, err := io.ReadAll(res.Body)
		helper.Must(err)

		helper.Must(res.Body.Close())

		r := result{}
		helper.Must(json.Unmarshal(b, &r))

		if res.StatusCode != tc.wantStatus {
			t.Errorf("expected status: %d, got %d", tc.wantStatus, res.StatusCode)
		}

		if r.Path != tc.wantPath {
			t.Errorf("expected path: %q, got: %q", tc.wantPath, r.Path)
		}
	}
}

func TestHTTPServer_OriginVsURL(t *testing.T) {
	client := newClient()

	configPath := "testdata/integration/url/"

	type expectation struct {
		Path  string
		Query url.Values
	}

	type testCase struct {
		file string
		exp  expectation
	}

	for _, tc := range []testCase{
		{"01_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"x": []string{"y"},
			},
		}},
		{"02_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"a": []string{"A"},
			},
		}},
		{"03_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"a": []string{"A"},
				"x": []string{"y"},
			},
		}},
		{"04_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"a": []string{"A"},
				"x": []string{"y"},
			},
		}},
		{"05_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"a": []string{"A"},
				"x": []string{"y"},
			},
		}},
		{"06_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"a": []string{"A"},
				"x": []string{"y"},
			},
		}},
	} {
		t.Run("File "+tc.file, func(subT *testing.T) {
			helper := test.New(subT)

			shutdown, _ := newCouper(path.Join(configPath, tc.file), helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080", nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_TrailingSlash(t *testing.T) {
	client := newClient()

	conf := "testdata/integration/endpoint_eval/11_couper.hcl"

	type expectation struct {
		Path string
	}

	type testCase struct {
		path string
		exp  expectation
	}

	for _, tc := range []testCase{
		{"/path", expectation{
			Path: "/path",
		}},
		{"/path/", expectation{
			Path: "/path/",
		}},
	} {
		t.Run("TrailingSlash "+tc.path, func(subT *testing.T) {
			helper := test.New(subT)
			shutdown, _ := newCouper(conf, helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_DynamicRequest(t *testing.T) {
	client := newClient()

	configFile := "testdata/integration/endpoint_eval/13_couper.hcl"
	shutdown, _ := newCouper(configFile, test.New(t))
	defer shutdown()

	type expectation struct {
		Body    string
		Headers http.Header
		Method  string
		Path    string
		Query   url.Values
	}

	type testCase struct {
		exp expectation
	}

	for _, tc := range []testCase{
		{expectation{
			Body:   "body",
			Method: "PUT",
			Path:   "/anything",
			Query: url.Values{
				"q": []string{"query"},
			},
			Headers: http.Header{
				"Content-Length": []string{"4"},
				"Content-Type":   []string{"text/plain"},
				"Test":           []string{"header"},
			},
		}},
	} {
		t.Run("Dynamic request", func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/?method=put", nil)
			helper.Must(err)

			req.Header.Set("Body", "body")
			req.Header.Set("Query", "query")
			req.Header.Set("Test", "header")

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_request_bodies(t *testing.T) {
	client := newClient()

	configFile := "testdata/integration/endpoint_eval/14_couper.hcl"
	shutdown, _ := newCouper(configFile, test.New(t))
	defer shutdown()

	type expectation struct {
		Body    string
		Args    url.Values
		Headers http.Header
		Method  string
	}

	type testCase struct {
		path              string
		clientPayload     string
		clientContentType string
		exp               expectation
	}

	for _, tc := range []testCase{
		{
			"/request/body",
			"",
			"",
			expectation{
				Body:   "foo",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"3"},
					"Content-Type":   []string{"text/plain"},
				},
			},
		},
		{
			"/request/body/ct",
			"",
			"",
			expectation{
				Body:   "foo",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"3"},
					"Content-Type":   []string{"application/foo"},
				},
			},
		},
		{
			"/request/json_body/null",
			"",
			"",
			expectation{
				Body:   "null",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"4"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/boolean",
			"",
			"",
			expectation{
				Body:   "true",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"4"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/boolean/ct",
			"",
			"",
			expectation{
				Body:   "true",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"4"},
					"Content-Type":   []string{"application/foo+json"},
				},
			},
		},
		{
			"/request/json_body/number",
			"",
			"",
			expectation{
				Body:   "1.2",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"3"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/string",
			"",
			"",
			expectation{
				Body:   `"föö"`,
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"7"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/object",
			"",
			"",
			expectation{
				Body:   `{"url":"http://...?foo&bar"}`,
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"28"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/object/html",
			"",
			"",
			expectation{
				Body:   `{"foo":"<p>bar</p>"}`,
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"20"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/array",
			"",
			"",
			expectation{
				Body:   "[0,1,2]",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"7"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/dyn",
			"true",
			"application/json",
			expectation{
				Body:   "true",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"4"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/dyn",
			"1.23",
			"application/json",
			expectation{
				Body:   "1.23",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"4"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/dyn",
			"\"ab\"",
			"application/json",
			expectation{
				Body:   "\"ab\"",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"4"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/dyn",
			"{\"a\":3,\"b\":[]}",
			"application/json",
			expectation{
				Body:   "{\"a\":3,\"b\":[]}",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"14"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/dyn",
			"[0,1]",
			"application/json",
			expectation{
				Body:   "[0,1]",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"5"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/form_body",
			"",
			"",
			expectation{
				Body: "",
				Args: url.Values{
					"foo": []string{"ab c"},
					"bar": []string{",:/"},
				},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"22"},
					"Content-Type":   []string{"application/x-www-form-urlencoded"},
				},
			},
		},
		{
			"/request/form_body/ct",
			"",
			"",
			expectation{
				Body:   "bar=%2C%3A%2F&foo=ab+c",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"22"},
					"Content-Type":   []string{"application/my-form-urlencoded"},
				},
			},
		},
		{
			"/request/form_body/dyn",
			"bar=%2C&foo=a",
			"application/x-www-form-urlencoded",
			expectation{
				Body: "",
				Args: url.Values{
					"foo": []string{"a"},
					"bar": []string{","},
				},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"13"},
					"Content-Type":   []string{"application/x-www-form-urlencoded"},
				},
			},
		},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodPost, "http://example.com:8080"+tc.path, strings.NewReader(tc.clientPayload))
			helper.Must(err)

			if tc.clientContentType != "" {
				req.Header.Set("Content-Type", tc.clientContentType)
			}

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_response_bodies(t *testing.T) {
	client := newClient()

	configFile := "testdata/integration/endpoint_eval/14_couper.hcl"
	shutdown, _ := newCouper(configFile, test.New(t))
	defer shutdown()

	type expectation struct {
		Body        string
		ContentType string
	}

	type testCase struct {
		path string
		exp  expectation
	}

	for _, tc := range []testCase{
		{
			"/response/body",
			expectation{
				Body:        "foo",
				ContentType: "text/plain",
			},
		},
		{
			"/response/body/ct",
			expectation{
				Body:        "foo",
				ContentType: "application/foo",
			},
		},
		{
			"/response/json_body/null",
			expectation{
				Body:        "null",
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/boolean",
			expectation{
				Body:        "true",
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/boolean/ct",
			expectation{
				Body:        "true",
				ContentType: "application/foo+json",
			},
		},
		{
			"/response/json_body/number",
			expectation{
				Body:        "1.2",
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/string",
			expectation{
				Body:        `"foo"`,
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/object",
			expectation{
				Body:        `{"foo":"bar"}`,
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/object/html",
			expectation{
				Body:        `{"foo":"<p>bar</p>"}`,
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/array",
			expectation{
				Body:        "[0,1,2]",
				ContentType: "application/json",
			},
		},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			res.Body.Close()

			if string(resBytes) != tc.exp.Body {
				subT.Errorf("%s: want: %s, got:%s", tc.path, tc.exp.Body, string(resBytes))
			}

			if ct := res.Header.Get("Content-Type"); ct != tc.exp.ContentType {
				subT.Errorf("%s: want: %s, got:%s", tc.path, tc.exp.ContentType, ct)
			}
		})
	}
}

func TestHTTPServer_Endpoint_Evaluation(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration/endpoint_eval/01_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	type expectation struct {
		Host, Origin, Path string
	}

	type testCase struct {
		reqPath string
		exp     expectation
	}

	// first traffic pins the origin (transport conf)
	for _, tc := range []testCase{
		{"/my-waffik/my.host.de/" + testBackend.Addr()[7:], expectation{
			Host:   "my.host.de",
			Origin: testBackend.Addr()[7:],
			Path:   "/anything",
		}},
		{"/my-respo/my.host.com/" + testBackend.Addr()[7:], expectation{
			Host:   "my.host.de",
			Origin: testBackend.Addr()[7:],
			Path:   "/anything",
		}},
	} {
		t.Run("_"+tc.reqPath, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.reqPath, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			jsonResult.Origin = res.Header.Get("X-Origin")

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_Endpoint_Response_FormQuery_Evaluation(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration/endpoint_eval/15_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodPost, "http://example.com:8080/req?foo=bar", strings.NewReader("s=abc123"))
	helper.Must(err)
	req.Header.Set("User-Agent", "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	type Expectation struct {
		FormBody url.Values  `json:"form_body"`
		Headers  test.Header `json:"headers"`
		Method   string      `json:"method"`
		Query    url.Values  `json:"query"`
		URL      string      `json:"url"`
	}

	var jsonResult Expectation
	err = json.Unmarshal(resBytes, &jsonResult)
	if err != nil {
		t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
	}

	delete(jsonResult.Headers, "couper-request-id")

	exp := Expectation{
		Method: http.MethodPost,
		FormBody: map[string][]string{
			"s": {"abc123"},
		},
		Headers: map[string]string{
			"content-length": "8",
			"content-type":   "application/x-www-form-urlencoded",
		},
		Query: map[string][]string{
			"foo": {"bar"},
		},
		URL: "http://example.com:8080/req?foo=bar",
	}
	if !reflect.DeepEqual(jsonResult, exp) {
		t.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", exp, jsonResult, string(resBytes))
	}
}

func TestHTTPServer_Endpoint_Response_JSONBody_Evaluation(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration/endpoint_eval/15_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/req?foo=bar", strings.NewReader(`{"data": true}`))
	helper.Must(err)
	req.Header.Set("User-Agent", "")
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	type Expectation struct {
		JSONBody map[string]interface{} `json:"json_body"`
		Headers  test.Header            `json:"headers"`
		Method   string                 `json:"method"`
		Query    url.Values             `json:"query"`
		URL      string                 `json:"url"`
	}

	var jsonResult Expectation
	err = json.Unmarshal(resBytes, &jsonResult)
	if err != nil {
		t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
	}

	delete(jsonResult.Headers, "couper-request-id")

	exp := Expectation{
		Method: http.MethodGet,
		JSONBody: map[string]interface{}{
			"data": true,
		},
		Headers: map[string]string{
			"content-length": "14",
			"content-type":   "application/json",
		},
		Query: map[string][]string{
			"foo": {"bar"},
		},
		URL: "http://example.com:8080/req?foo=bar",
	}
	if !reflect.DeepEqual(jsonResult, exp) {
		t.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", exp, jsonResult, string(resBytes))
	}
}

func TestHTTPServer_Endpoint_Response_JSONBody_Array_Evaluation(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration/endpoint_eval/15_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	helper := test.New(t)

	content := `[1, 2, {"data": true}]`

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/req?foo=bar", strings.NewReader(content))
	helper.Must(err)
	req.Header.Set("User-Agent", "")
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	type Expectation struct {
		JSONBody interface{} `json:"json_body"`
		Headers  test.Header `json:"headers"`
		Method   string      `json:"method"`
		Query    url.Values  `json:"query"`
		URL      string      `json:"url"`
	}

	var jsonResult Expectation
	err = json.Unmarshal(resBytes, &jsonResult)
	if err != nil {
		t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
	}

	delete(jsonResult.Headers, "couper-request-id")

	exp := Expectation{
		Method: http.MethodGet,
		JSONBody: []interface{}{
			1,
			2,
			map[string]interface{}{
				"data": true,
			},
		},
		Headers: map[string]string{
			"content-length": strconv.Itoa(len(content)),
			"content-type":   "application/json",
		},
		Query: map[string][]string{
			"foo": {"bar"},
		},
		URL: "http://example.com:8080/req?foo=bar",
	}

	if fmt.Sprint(jsonResult) != fmt.Sprint(exp) {
		t.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", exp, jsonResult, string(resBytes))
	}
}

func TestHTTPServer_AcceptingForwardedURL(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/settings/05_couper.hcl")
	shutdown, hook := newCouper(confPath, test.New(t))
	defer shutdown()

	type expectation struct {
		Protocol string `json:"protocol"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Origin   string `json:"origin"`
		URL      string `json:"url"`
	}

	type testCase struct {
		name             string
		header           http.Header
		exp              expectation
		wantAccessLogURL string
	}

	for _, tc := range []testCase{
		{
			"no proto, host, or port",
			http.Header{},
			expectation{
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Origin:   "http://localhost:8080",
				URL:      "http://localhost:8080/path",
			},
			"http://localhost:8080/path",
		},
		{
			"port, no proto, no host",
			http.Header{
				"X-Forwarded-Port": []string{"8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "localhost",
				Port:     8081,
				Origin:   "http://localhost:8081",
				URL:      "http://localhost:8081/path",
			},
			"http://localhost:8081/path",
		},
		{
			"proto, no host, no port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
			},
			expectation{
				Protocol: "https",
				Host:     "localhost",
				Port:     443,
				Origin:   "https://localhost",
				URL:      "https://localhost/path",
			},
			"https://localhost/path",
		},
		{
			"proto, host, no port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     443,
				Origin:   "https://www.example.com",
				URL:      "https://www.example.com/path",
			},
			"https://www.example.com/path",
		},
		{
			"proto, host with port, no port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com:8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "https://www.example.com:8443",
				URL:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
		},
		{
			"proto, port, no host",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Port":  []string{"8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "localhost",
				Port:     8443,
				Origin:   "https://localhost:8443",
				URL:      "https://localhost:8443/path",
			},
			"https://localhost:8443/path",
		},
		{
			"host, port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com"},
				"X-Forwarded-Port": []string{"8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8081,
				Origin:   "http://www.example.com:8081",
				URL:      "http://www.example.com:8081/path",
			},
			"http://www.example.com:8081/path",
		},
		{
			"host with port, port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com:8081"},
				"X-Forwarded-Port": []string{"8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8081,
				Origin:   "http://www.example.com:8081",
				URL:      "http://www.example.com:8081/path",
			},
			"http://www.example.com:8081/path",
		},
		{
			"host with port, different port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com:8081"},
				"X-Forwarded-Port": []string{"8082"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8082,
				Origin:   "http://www.example.com:8082",
				URL:      "http://www.example.com:8082/path",
			},
			"http://www.example.com:8082/path",
		},
		{
			"host, no port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8080,
				Origin:   "http://www.example.com:8080",
				URL:      "http://www.example.com:8080/path",
			},
			"http://www.example.com:8080/path",
		},
		{
			"host with port, no proto, no port",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com:8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8081,
				Origin:   "http://www.example.com:8081",
				URL:      "http://www.example.com:8081/path",
			},
			"http://www.example.com:8081/path",
		},
		{
			"proto, host, port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com"},
				"X-Forwarded-Port":  []string{"8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "https://www.example.com:8443",
				URL:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
		},
		{
			"proto, host with port, port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com:8443"},
				"X-Forwarded-Port":  []string{"8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "https://www.example.com:8443",
				URL:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
		},
		{
			"proto, host with port, different port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com:8443"},
				"X-Forwarded-Port":  []string{"9443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     9443,
				Origin:   "https://www.example.com:9443",
				URL:      "https://www.example.com:9443/path",
			},
			"https://www.example.com:9443/path",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/path", nil)
			helper.Must(err)
			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.exp, jsonResult, string(resBytes))
			}

			logURL := getAccessLogURL(hook)
			if logURL != tc.wantAccessLogURL {
				subT.Errorf("Expected URL: %q, actual: %q", tc.wantAccessLogURL, logURL)
			}
		})
	}
}

func TestHTTPServer_XFH_AcceptingForwardedURL(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/settings/06_couper.hcl")
	shutdown, hook := newCouper(confPath, test.New(t))
	defer shutdown()

	type expectation struct {
		Protocol string `json:"protocol"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Origin   string `json:"origin"`
		URL      string `json:"url"`
	}

	type testCase struct {
		name             string
		header           http.Header
		exp              expectation
		wantAccessLogURL string
	}

	for _, tc := range []testCase{
		{
			"no proto, host, or port",
			http.Header{},
			expectation{
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Origin:   "http://localhost:8080",
				URL:      "http://localhost:8080/path",
			},
			"http://localhost:8080/path",
		},
		{
			"port, no proto, no host",
			http.Header{
				"X-Forwarded-Port": []string{"8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "localhost",
				Port:     8081,
				Origin:   "http://localhost:8081",
				URL:      "http://localhost:8081/path",
			},
			"http://localhost:8081/path",
		},
		{
			"proto, no host, no port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
			},
			expectation{
				Protocol: "https",
				Host:     "localhost",
				Port:     443,
				Origin:   "https://localhost",
				URL:      "https://localhost/path",
			},
			"https://localhost/path",
		},
		{
			"proto, host, no port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     443,
				Origin:   "https://www.example.com",
				URL:      "https://www.example.com/path",
			},
			"https://www.example.com/path",
		},
		{
			"proto, host with port, no port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com:8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     443,
				Origin:   "https://www.example.com",
				URL:      "https://www.example.com/path",
			},
			"https://www.example.com/path",
		},
		{
			"proto, port, no host",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Port":  []string{"8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "localhost",
				Port:     8443,
				Origin:   "https://localhost:8443",
				URL:      "https://localhost:8443/path",
			},
			"https://localhost:8443/path",
		},
		{
			"host, port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com"},
				"X-Forwarded-Port": []string{"8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8081,
				Origin:   "http://www.example.com:8081",
				URL:      "http://www.example.com:8081/path",
			},
			"http://www.example.com:8081/path",
		},
		{
			"host with port, port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com:8081"},
				"X-Forwarded-Port": []string{"8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8081,
				Origin:   "http://www.example.com:8081",
				URL:      "http://www.example.com:8081/path",
			},
			"http://www.example.com:8081/path",
		},
		{
			"host with port, different port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com:8081"},
				"X-Forwarded-Port": []string{"8082"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8082,
				Origin:   "http://www.example.com:8082",
				URL:      "http://www.example.com:8082/path",
			},
			"http://www.example.com:8082/path",
		},
		{
			"host, no port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8080,
				Origin:   "http://www.example.com:8080",
				URL:      "http://www.example.com:8080/path",
			},
			"http://www.example.com:8080/path",
		},
		{
			"host with port, no proto, no port",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com:8081"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8080,
				Origin:   "http://www.example.com:8080",
				URL:      "http://www.example.com:8080/path",
			},
			"http://www.example.com:8080/path",
		},
		{
			"proto, host, port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com"},
				"X-Forwarded-Port":  []string{"8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "https://www.example.com:8443",
				URL:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
		},
		{
			"proto, host with port, port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com:8443"},
				"X-Forwarded-Port":  []string{"8443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "https://www.example.com:8443",
				URL:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
		},
		{
			"proto, host with port, different port",
			http.Header{
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Host":  []string{"www.example.com:8443"},
				"X-Forwarded-Port":  []string{"9443"},
			},
			expectation{
				Protocol: "https",
				Host:     "www.example.com",
				Port:     9443,
				Origin:   "https://www.example.com:9443",
				URL:      "https://www.example.com:9443/path",
			},
			"https://www.example.com:9443/path",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/path", nil)
			helper.Must(err)
			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.exp, jsonResult, string(resBytes))
			}

			logURL := getAccessLogURL(hook)
			if logURL != tc.wantAccessLogURL {
				subT.Errorf("Expected URL: %q, actual: %q", tc.wantAccessLogURL, logURL)
			}
		})
	}
}

func TestHTTPServer_BackendProbes(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	confPath := path.Join("testdata/integration/config/14_couper.hcl")
	shutdown, _ := newCouper(confPath, helper)
	defer shutdown()

	type testCase struct {
		name   string
		path   string
		expect string
	}

	time.Sleep(2 * time.Second)
	healthyJSON := `{"error":"","healthy":true,"state":"healthy"}`

	for _, tc := range []testCase{
		{
			"unknown backend",
			"/unknown",
			`null`,
		},
		{
			"healthy backend",
			"/healthy/default",
			healthyJSON,
		},
		{
			"healthy backend w/ expected_status",
			"/healthy/expected_status",
			healthyJSON,
		},
		{
			"healthy backend w/ expected_text",
			"/healthy/expected_text",
			healthyJSON,
		},
		{
			"healthy backend w/ path",
			"/healthy/path",
			healthyJSON,
		},
		{
			"healthy backend w/ headers",
			"/healthy/headers",
			healthyJSON,
		},
		{
			"healthy backend w/ fallback ua header",
			"/healthy/ua-header",
			healthyJSON,
		},
		{
			"healthy backend: check does not follow Location",
			"/healthy/no_follow_redirect",
			healthyJSON,
		},
		{
			"unhealthy backend: timeout",
			"/unhealthy/timeout",
			`{"error":"backend error: connecting to unhealthy_timeout '1.2.3.4' failed: i/o timeout","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend: unexpected status code",
			"/unhealthy/bad_status",
			`{"error":"unexpected status code: 404","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend w/ expected_status: unexpected status code",
			"/unhealthy/bad_expected_status",
			`{"error":"unexpected status code: 200","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend w/ expected_text: unexpected text",
			"/unhealthy/bad_expected_text",
			`{"error":"unexpected text","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend: unexpected status code",
			"/unhealthy/bad_status",
			`{"error":"unexpected status code: 404","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend w/ path: unexpected status code",
			"/unhealthy/bad_path",
			`{"error":"unexpected status code: 404","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend w/ headers: unexpected text",
			"/unhealthy/headers",
			`{"error":"unexpected text","healthy":false,"state":"unhealthy"}`,
		},
		{
			"unhealthy backend: does not follow location",
			"/unhealthy/no_follow_redirect",
			`{"error":"unexpected status code: 302","healthy":false,"state":"unhealthy"}`,
		},
		{
			"backend error: timeout but threshold not reached",
			"/failing",
			`{"error":"backend error: connecting to failing '1.2.3.4' failed: i/o timeout","healthy":true,"state":"failing"}`,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			h.Must(err)

			res, err := client.Do(req)
			h.Must(err)

			b, _ := io.ReadAll(res.Body)
			body := string(b)
			h.Must(res.Body.Close())

			if body != tc.expect {
				t.Errorf("%s: Unexpected states:\n\tWant: %s\n\tGot:  %s", tc.name, tc.expect, body)
			}
		})
	}
}

func TestHTTPServer_backend_requests_variables(t *testing.T) {
	client := newClient()

	ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer ResourceOrigin.Close()

	confPath := path.Join("testdata/integration/endpoint_eval/18_couper.hcl")
	shutdown, hook, err := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"rsOrigin": ResourceOrigin.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()

	type expectation struct {
		Method   string                 `json:"method"`
		Protocol string                 `json:"protocol"`
		Host     string                 `json:"host"`
		Port     int64                  `json:"port"`
		Path     string                 `json:"path"`
		Query    map[string][]string    `json:"query"`
		Origin   string                 `json:"origin"`
		URL      string                 `json:"url"`
		Body     string                 `json:"body"`
		JSONBody map[string]interface{} `json:"json_body"`
		FormBody map[string][]string    `json:"form_body"`
	}

	type testCase struct {
		name   string
		relURL string
		header http.Header
		body   io.Reader
		exp    expectation
	}

	helper := test.New(t)
	resourceOrigin, perr := url.Parse(ResourceOrigin.URL)
	helper.Must(perr)

	port, _ := strconv.ParseInt(resourceOrigin.Port(), 10, 64)

	for _, tc := range []testCase{
		{
			"body",
			"/body",
			http.Header{},
			strings.NewReader(`abcd1234`),
			expectation{
				Method:   http.MethodPost,
				Protocol: resourceOrigin.Scheme,
				Host:     resourceOrigin.Hostname(),
				Port:     port,
				Path:     "/resource",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   ResourceOrigin.URL,
				URL:      ResourceOrigin.URL + "/resource?foo=bar",
				Body:     "abcd1234",
				JSONBody: map[string]interface{}{},
				FormBody: map[string][]string{},
			},
		},
		{
			"json_body",
			"/json_body",
			http.Header{"Content-Type": []string{"application/json"}},
			strings.NewReader(`{"s":"abcd1234"}`),
			expectation{
				Method:   http.MethodPost,
				Protocol: resourceOrigin.Scheme,
				Host:     resourceOrigin.Hostname(),
				Port:     port,
				Path:     "/resource",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   ResourceOrigin.URL,
				URL:      ResourceOrigin.URL + "/resource?foo=bar",
				Body:     `{"s":"abcd1234"}`,
				JSONBody: map[string]interface{}{"s": "abcd1234"},
				FormBody: map[string][]string{},
			},
		},
		{
			"form_body",
			"/form_body",
			http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}},
			strings.NewReader(`s=abcd1234`),
			expectation{
				Method:   http.MethodPost,
				Protocol: resourceOrigin.Scheme,
				Host:     resourceOrigin.Hostname(),
				Port:     port,
				Path:     "/resource",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   ResourceOrigin.URL,
				URL:      ResourceOrigin.URL + "/resource?foo=bar",
				Body:     `s=abcd1234`,
				JSONBody: map[string]interface{}{},
				FormBody: map[string][]string{"s": {"abcd1234"}},
			},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodPost, "http://localhost:8080"+tc.relURL, tc.body)
			h.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			h.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			h.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("%s: unmarshal json: %v: got:\n%s", tc.name, err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("%s\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.name, tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_request_variables(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration/endpoint_eval/19_couper.hcl")
	shutdown, hook := newCouper(confPath, test.New(t))
	defer shutdown()

	type expectation struct {
		Method   string                 `json:"method"`
		Protocol string                 `json:"protocol"`
		Host     string                 `json:"host"`
		Port     int64                  `json:"port"`
		Path     string                 `json:"path"`
		Query    map[string][]string    `json:"query"`
		Origin   string                 `json:"origin"`
		URL      string                 `json:"url"`
		Body     string                 `json:"body"`
		JSONBody map[string]interface{} `json:"json_body"`
		FormBody map[string][]string    `json:"form_body"`
	}

	type testCase struct {
		name   string
		relURL string
		header http.Header
		body   io.Reader
		exp    expectation
	}

	for _, tc := range []testCase{
		{
			"body",
			"/body?foo=bar",
			http.Header{},
			strings.NewReader(`abcd1234`),
			expectation{
				Method:   "POST",
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Path:     "/body",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   "http://localhost:8080",
				URL:      "http://localhost:8080/body?foo=bar",
				Body:     "abcd1234",
				JSONBody: map[string]interface{}{},
				FormBody: map[string][]string{},
			},
		},
		{
			"json_body",
			"/json_body?foo=bar",
			http.Header{"Content-Type": []string{"application/json"}},
			strings.NewReader(`{"s":"abcd1234"}`),
			expectation{
				Method:   "POST",
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Path:     "/json_body",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   "http://localhost:8080",
				URL:      "http://localhost:8080/json_body?foo=bar",
				Body:     `{"s":"abcd1234"}`,
				JSONBody: map[string]interface{}{"s": "abcd1234"},
				FormBody: map[string][]string{},
			},
		},
		{
			"form_body",
			"/form_body?foo=bar",
			http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}},
			strings.NewReader(`s=abcd1234`),
			expectation{
				Method:   "POST",
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Path:     "/form_body",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   "http://localhost:8080",
				URL:      "http://localhost:8080/form_body?foo=bar",
				Body:     `s=abcd1234`,
				JSONBody: map[string]interface{}{},
				FormBody: map[string][]string{"s": {"abcd1234"}},
			},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodPost, "http://localhost:8080"+tc.relURL, tc.body)
			helper.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("%s: unmarshal json: %v: got:\n%s", tc.name, err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("%s\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.name, tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestOpenAPIValidateConcurrentRequests(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/validation/01_couper.hcl", helper)
	defer shutdown()

	req1, err := http.NewRequest(http.MethodGet, "http://example.com:8080/anything", nil)
	helper.Must(err)
	req2, err := http.NewRequest(http.MethodGet, "http://example.com:8080/pdf", nil)
	helper.Must(err)

	var res1, res2 *http.Response
	var err1, err2 error
	waitCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-waitCh // blocks
		res1, err1 = client.Do(req1)
	}()
	go func() {
		defer wg.Done()
		<-waitCh // blocks
		res2, err2 = client.Do(req2)
	}()

	close(waitCh) // triggers reqs
	wg.Wait()

	helper.Must(err1)
	helper.Must(err2)

	if res1.StatusCode != 200 {
		t.Errorf("Expected status %d for response1; got: %d", 200, res1.StatusCode)
	}
	if res2.StatusCode != 502 {
		t.Errorf("Expected status %d for response2; got: %d", 502, res2.StatusCode)
	}
}

func TestOpenAPIValidateRequestResponseBuffer(t *testing.T) {
	helper := test.New(t)

	content := `{ "prop": true }`
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		helper.Must(err)
		if string(b) != content {
			t.Errorf("origin: expected same content")
		}
		rw.Header().Set("Content-Type", "application/json")
		_, err = rw.Write([]byte(content))
		helper.Must(err)
	}))
	defer origin.Close()

	shutdown, _ := newCouper("testdata/integration/validation/02_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/buffer", bytes.NewBufferString(content))
	helper.Must(err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin.URL)

	res, err := test.NewHTTPClient().Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusOK, got: %s", res.Status)
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	helper.Must(res.Body.Close())

	if string(b) != content {
		t.Error("expected same body content")
	}
}

func TestConfigBodyContent(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	expiredOrigin, selfSigned := test.NewExpiredBackend()
	defer expiredOrigin.Close()

	expiredCert, err := os.CreateTemp(os.TempDir(), "expired.pem")
	helper.Must(err)

	_, err = expiredCert.Write(selfSigned.CACertificate.Certificate)
	helper.Must(err)
	helper.Must(expiredCert.Close())

	defer os.RemoveAll(expiredCert.Name())

	shutdown, _, err := newCouperWithTemplate("testdata/integration/config/01_couper.hcl", helper, map[string]interface{}{
		"expiredOrigin": expiredOrigin.Addr(),
		"caFile":        expiredCert.Name(),
	})
	helper.Must(err)
	defer shutdown()

	// default port changed in config
	req, err := http.NewRequest(http.MethodGet, "http://time.out:8090/", nil)
	helper.Must(err)

	// 2s timeout in config
	ctx, cancel := context.WithDeadline(req.Context(), time.Now().Add(time.Second*10))
	defer cancel()
	*req = *req.Clone(ctx)
	defer func() {
		if e := ctx.Err(); e != nil {
			t.Error("Expected used config timeout instead of deadline timer")
		}
	}()

	_, err = client.Do(req)
	helper.Must(err)

	// disabled cert check in config
	req, err = http.NewRequest(http.MethodGet, "http://time.out:8090/expired/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK with disabled certificate validation, got: %q", res.Status)
	}
}

func TestConfigBodyContentBackends(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/config/02_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path   string
		header http.Header
		query  url.Values
	}

	for _, tc := range []testCase{
		{"/anything", http.Header{"Foo": []string{"4"}}, url.Values{"bar": []string{"3", "4"}}},
		{"/get", http.Header{"Foo": []string{"1", "3"}}, url.Values{"bar": []string{"1", "4"}}},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)
			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Errorf("%q: expected Status OK, got: %d", tc.path, res.StatusCode)
			}

			b, err := io.ReadAll(res.Body)
			helper.Must(err)

			type payload struct {
				Query url.Values
			}
			var p payload
			helper.Must(json.Unmarshal(b, &p))

			for k, v := range tc.header {
				if !reflect.DeepEqual(res.Header[k], v) {
					subT.Errorf("Expected Header %q value: %v, got: %v", k, v, res.Header[k])
				}
			}

			for k, v := range tc.query {
				if !reflect.DeepEqual(p.Query[k], v) {
					subT.Errorf("Expected Query %q value: %v, got: %v", k, v, p.Query[k])
				}
			}
		})
	}
}

func TestConfigBodyContentAccessControl(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/03_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path       string
		header     http.Header
		status     int
		ct         string
		wantErrLog string
	}

	for _, tc := range []testCase{
		{"/v1", http.Header{"Auth": []string{"ba1"}}, http.StatusOK, "application/json", ""},
		{"/v2", http.Header{"Authorization": []string{"Basic OmFzZGY="}, "Auth": []string{"ba1", "ba2"}}, http.StatusOK, "application/json", ""}, // minimum ':'
		{"/v2", http.Header{}, http.StatusUnauthorized, "application/json", "access control error: ba1: credentials required"},
		{"/v3", http.Header{}, http.StatusOK, "application/json", ""},
		{"/status", http.Header{}, http.StatusOK, "application/json", ""},
		{"/superadmin", http.Header{"Authorization": []string{"Basic OmFzZGY="}, "Auth": []string{"ba1", "ba4"}}, http.StatusOK, "application/json", ""},
		{"/superadmin", http.Header{}, http.StatusUnauthorized, "application/json", "access control error: ba1: credentials required"},
		{"/ba5", http.Header{"Authorization": []string{"Basic VVNSOlBXRA=="}, "X-Ba-User": []string{"USR"}}, http.StatusOK, "application/json", ""},
		{"/v4", http.Header{}, http.StatusUnauthorized, "text/html", "access control error: ba1: credentials required"},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			if val := tc.header.Get("Authorization"); val != "" {
				req.Header.Set("Authorization", val)
			}

			res, err := client.Do(req)
			helper.Must(err)
			// t.Errorf(">>> %#v", res.Header)

			message := getFirstAccessLogMessage(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					subT.Errorf("Expected error log: %q, actual: %#v", tc.wantErrLog, message)
				}
			} else {
				if message != tc.wantErrLog {
					subT.Errorf("Expected error log message: %q, actual: %#v", tc.wantErrLog, message)
				}
			}

			if res.StatusCode != tc.status {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.path, tc.status, res.StatusCode)
			}

			if ct := res.Header.Get("Content-Type"); ct != tc.ct {
				subT.Fatalf("%q: expected content-type: %q, got: %q", tc.path, tc.ct, ct)
			}

			if tc.ct == "text/html" {
				return
			}

			b, err := io.ReadAll(res.Body)
			helper.Must(err)

			type payload struct {
				Headers http.Header
			}
			var p payload
			helper.Must(json.Unmarshal(b, &p))

			for k, v := range tc.header {
				if _, ok := p.Headers[k]; !ok {
					subT.Errorf("Expected header %q, got nothing", k)
					break
				}
				if !reflect.DeepEqual(p.Headers[k], v) {
					subT.Errorf("Expected header %q value: %v, got: %v", k, v, p.Headers[k])
				}
			}
		})
	}
}

func TestAPICatchAll(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/03_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name       string
		path       string
		method     string
		header     http.Header
		status     int
		wantErrLog string
	}

	for _, tc := range []testCase{
		{"exists, authorized", "/v5/exists", http.MethodGet, http.Header{"Authorization": []string{"Basic OmFzZGY="}}, http.StatusOK, ""},
		{"exists, unauthorized", "/v5/exists", http.MethodGet, http.Header{}, http.StatusUnauthorized, "access control error: ba1: credentials required"},
		{"exists, CORS pre-flight", "/v5/exists", http.MethodOptions, http.Header{"Origin": []string{"https://www.example.com"}, "Access-Control-Request-Method": []string{"POST"}}, http.StatusNoContent, ""},
		{"not-exist, authorized", "/v5/not-exist", http.MethodGet, http.Header{"Authorization": []string{"Basic OmFzZGY="}}, http.StatusNotFound, "route not found error"},
		{"not-exist, unauthorized", "/v5/not-exist", http.MethodGet, http.Header{}, http.StatusUnauthorized, "access control error: ba1: credentials required"},
		{"not-exist, non-standard method, authorized", "/v5/not-exist", "BREW", http.Header{"Authorization": []string{"Basic OmFzZGY="}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"not-exist, non-standard method, unauthorized", "/v5/not-exist", "BREW", http.Header{}, http.StatusUnauthorized, "access control error: ba1: credentials required"},
		{"not-exist, CORS pre-flight", "/v5/not-exist", http.MethodOptions, http.Header{"Origin": []string{"https://www.example.com"}, "Access-Control-Request-Method": []string{"POST"}}, http.StatusNoContent, ""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(tc.method, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			req.Header = tc.header

			res, err := client.Do(req)
			helper.Must(err)

			message := getFirstAccessLogMessage(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					subT.Errorf("Expected error log: %q, actual: %#v", tc.wantErrLog, message)
				}
			} else {
				if message != tc.wantErrLog {
					subT.Errorf("Expected error log message: %q, actual: %#v", tc.wantErrLog, message)
				}
			}

			if res.StatusCode != tc.status {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.path, tc.status, res.StatusCode)
			}
		})
	}
}

func Test_LoadAccessControl(t *testing.T) {
	// Tests the config load with ACs and "error_handler" blocks...
	backend := test.NewBackend()
	defer backend.Close()

	shutdown, _, err := newCouperWithTemplate("testdata/integration/config/07_couper.hcl", test.New(t), map[string]interface{}{
		"asOrigin": backend.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	test.WaitForOpenPort(8080)
	shutdown()
}

func TestJWTAccessControl(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/03_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name       string
		path       string
		header     http.Header
		body       string
		status     int
		expPerm    string
		wantErrLog string
	}

	tokenRequest, reqerr := http.NewRequest(http.MethodGet, "http://back.end:8080/jwt/create?type=ECDSAToken", nil)
	if reqerr != nil {
		t.Fatal(reqerr)
	}
	tokenResponse, resperr := client.Do(tokenRequest)
	if reqerr != nil {
		t.Fatal(resperr)
	}
	bytes, _ := io.ReadAll(tokenResponse.Body)
	localToken := string(bytes)

	// RSA tokens created with server/testdata/integration/files/pkcs8.key
	// ECDSA tokens created with server/testdata/integration/files/ecdsa.key
	rsaToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.AZ0gZVqPe9TjjjJO0GnlTvERBXhPyxW_gTn050rCoEkseFRlp4TYry7WTQ7J4HNrH3btfxaEQLtTv7KooVLXQyMDujQbKU6cyuYH6MZXaM0Co3Bhu0awoX-2GVk997-7kMZx2yvwIR5ypd1CERIbNs5QcQaI4sqx_8oGrjO5ZmOWRqSpi4Mb8gJEVVccxurPu65gPFq9esVWwTf4cMQ3GGzijatnGDbRWs_igVGf8IAfmiROSVd17fShQtfthOFd19TGUswVAleOftC7-DDeJgAK8Un5xOHGRjv3ypK_6ZLRonhswaGXxovE0kLq4ZSzumQY2hOFE6x_BbrR1WKtGw"
	hmacToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwic2NvcGUiOiJmb28gYmFyIiwiaWF0IjoxNTE2MjM5MDIyfQ.7wz7Z7IajfEpwYayfshag6tQVS0e0zZJyjAhuFC0L-E"
	ecdsaToken := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVzMjU2In0.eyJzdWIiOjEyMzQ1Njc4OTB9.jXsNtPUXxBi8Bz2i2Maj9lzbB1ebQDmz8TU6GSs6G0yzq9YguXm_HQuwsg4ZTbPER3bpXH_cxz9eEZHUBXfWzw"
	ecdsaToken2 := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVzMjU2LWNydi14LXkifQ.eyJzdWIiOjEyMzQ1Njc4OTB9.4-uNC6KGkSY1YYAmGoR-naUu2-Rxo6HSzEkecb7Ua9FVkif0X2gC55DpPU06_HH-yfK-dFozLwzuV2AT6ouOIg"

	for _, tc := range []testCase{
		{"no token", "/jwt", http.Header{}, "", http.StatusUnauthorized, "", "access control error: JWTToken: token required"},
		{"expired token", "/jwt", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjEyMzQ1Njc4OSwic2NvcGUiOlsiZm9vIiwiYmFyIl19.W2ziH_V33JkOA5ttQhzWN96RqxFydmx7GHY6G__U9HM"}}, "", http.StatusForbidden, "", "access control error: JWTToken: Token is expired"},
		{"valid token", "/jwt", http.Header{"Authorization": []string{"Bearer " + hmacToken}}, "", http.StatusOK, `["foo","bar"]`, ""},
		{"RSA JWT", "/jwt/rsa", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"RSA JWT PKCS1", "/jwt/rsa/pkcs1", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"RSA JWT PKCS8", "/jwt/rsa/pkcs8", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"RSA JWT bad algorithm", "/jwt/rsa/bad", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusForbidden, "", "access control error: RSATokenWrongAlgorithm: signing method RS256 is invalid"},
		{"local RSA JWKS without kid", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEyMzQ1Njc4OTB9.V9skZUql-mHqwOzVdzamqAOWSx8fjEA-6py0nfxLRSl7h1bQvqUCWMZUAkMJK6RuJ3y5YAr8ZBXZsh4rwABp_3hitQitMXnV6nr5qfzVDE9-mdS4--Bj46-JlkHacNcK24qlnn_EXGJlzCj6VFgjObSy6geaTY9iDVF6EzjZkxc1H75XRlNYAMu-0KCGfKdte0qASeBKrWnoFNEpnXZ_jhqRRNVkaSBj7_HPXD6oPqKBQf6Jh6fGgdz6q4KNL-t-Qa2_eKc8tkrYNdTdxco-ufmmLiUQ_MzRAqowHb2LdsFJP9rN2QT8MGjRXqGvkCd0EsLfqAeCPkTXs1kN8LGlvw"}}, "", http.StatusForbidden, "", `access control error: JWKS: no matching RS256 JWK for kid ""`},
		{"local RSA JWKS with unsupported kid", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer eyJraWQiOiJyczI1Ni11bnN1cHBvcnRlZCIsImFsZyI6IlJTMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.wx1MkMgJhh6gnOvvrnnkRpEUDe-0KpKWw9ZIfDVHtGkuL46AktBgfbaW1ttB78wWrIW9OPfpLqKwkPizwfShoXKF9qN-6TlhPSWIUh0_kBHEj7H4u45YZXH1Ha-r9kGzly1PmLx7gzxUqRpqYnwo0TzZSEr_a8rpfWaC0ZJl3CKARormeF3tzW_ARHnGUqck4VjPfX50Ot6B5nool6qmsCQLLmDECIKBDzZicqdeWH7JPvRZx45R5ZHJRQpD3Z2iqVIF177Wj1C8q75Gxj2PXziIVKplmIUrKN-elYj3kBtJkDFneb384FPLuzsQZOR6HQmKXG2nA1WOfsblJSz3FA"}}, "", http.StatusForbidden, "", `access control error: JWKS: no matching RS256 JWK for kid "rs256-unsupported"`},
		{"local RSA JWKS with non-parsable cert", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer eyJraWQiOiJyczI1Ni13cm9uZy1jZXJ0IiwiYWxnIjoiUlMyNTYiLCJ0eXAiOiJKV1QifQ.eyJzdWIiOjEyMzQ1Njc4OTB9.n--6mjzfnPKbaYAquBK3v6gsbmvEofSprk3jwWGSKPdDt2VpVOe8ZNtGhJj_3f1h86-wg-gEQT5GhJmsI47X9MJ70j74dqhXUF6w4782OljstP955whuSM9hJAIvUw_WV1sqtkiESA-CZiNJIBydL5YzV2nO3gfEYdy9EdMJ2ykGLRBajRxhShxsfaZykFKvvWpy1LbUc-gfRZ4q8Hs9B7b_9RGdbpRwBtwiqPPzhjC5O86vk7ZoiG9Gq7pg52yEkLqdN4a5QkfP8nNeTTMAsqPQL1-1TAC7rIGekoUtoINRR-cewPpZ_E7JVxXvBVvPe3gX_2NzGtXkLg5QDt6RzQ"}}, "", http.StatusForbidden, "", `access control error: JWKS: no matching RS256 JWK for kid "rs256-wrong-cert"`},
		{"local RSA JWKS not found", "/jwks/rsa/not_found", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusForbidden, "", `access control error: JWKS_not_found: received no valid JWKs data: <nil>, status code 404`},
		{"local RSA JWKS", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"local RSA JWKS with scope", "/jwks/rsa/scope", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTAsInNjb3BlIjpbImZvbyIsImJhciJdfQ.IFqIF_9ELXl3A-oy52G0Sg5f34ah3araOxFboskEw110nXdb_-UuxCnG0naFVFje7xvNrGbJgVAbBRX1v1I_to4BR8RzvIh2hi5IgBmqclIYsYbVWlEhsvjBhFR2b90Rz0APUdfgHp-nvgLB13jxm8f4TRr4ZDnvUQdZp3vI5PMj9optEmlZvexkNLDQLrBvoGCfVHodZyPQMLNVKp0TXWksPT-bw0E7Lq1GeYe2eU0GwHx8fugo2-v44dfCp0RXYYG6bI_Z-U3KZpvdj05n2_UDgTJFFm4c5i9UjILvlO73QJpMNi5eBjerm2alTisSCoiCtfgIgVsM8yHoomgarg"}}, "", http.StatusOK, `["foo","bar"]`, ""},
		{"remote RSA JWKS x5c", "/jwks/rsa/remote", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"remote RSA JWKS x5c w/ backend", "/jwks/rsa/backend", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"remote RSA JWKS x5c w/ backendref", "/jwks/rsa/backendref", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, "", http.StatusOK, "", ""},
		{"remote RSA JWKS n, e", "/jwks/rsa/remote", http.Header{"Authorization": []string{"Bearer eyJraWQiOiJyczI1Ni1uZSIsImFsZyI6IlJTMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.aGOhlWQIZvnwoEZGDBYhkkEduIVa59G57x88L3fiLc1MuWbYS84nHEZnlPDuVJ3_BxdXr6-nZ8gpk1C9vfamDzkbvzbdcJ2FzmvAONm1II3_u5OTc6ZtpREDx9ohlIvkcOcalOUhQLqU5r2uik2bGSVV3vFDbqxQeuNzh49i3VgdtwoaryNYSzbg_Ki8dHiaFrWH-r2WCU08utqpFmNdr8oNw4Y5AYJdUW2aItxDbwJ6YLBJN0_6EApbXsNqiaNXkLws3cxMvczGKODyGGVCPENa-VmTQ41HxsXB-_rMmcnMw3_MjyIueWcjeP8BNvLYt1bKFWdU0NcYCkXvEqE4-g"}}, "", http.StatusOK, "", ""},
		{"token_value query", "/jwt/token_value_query?token=" + hmacToken, http.Header{}, "", http.StatusOK, `["foo","bar"]`, ""},
		{"token_value body", "/jwt/token_value_body", http.Header{"Content-Type": {"application/json"}}, `{"token":"` + hmacToken + `"}`, http.StatusOK, `["foo","bar"]`, ""},
		{"ECDSA JWT", "/jwt/ecdsa", http.Header{"Authorization": []string{"Bearer " + ecdsaToken}}, "", http.StatusOK, "", ""},
		{"ECDSA local JWT", "/jwt/ecdsa", http.Header{"Authorization": []string{"Bearer " + localToken}}, "", http.StatusOK, "", ""},
		{"ECDSA JWT PKCS8", "/jwt/ecdsa8", http.Header{"Authorization": []string{"Bearer " + ecdsaToken}}, "", http.StatusOK, "", ""},
		{"ECDSA JWT bad algorithm", "/jwt/ecdsa/bad", http.Header{"Authorization": []string{"Bearer " + ecdsaToken}}, "", http.StatusForbidden, "", "access control error: ECDSATokenWrongAlgorithm: signing method ES256 is invalid"},
		{"ECDSA JWKS with certificate: kid=es256", "/jwks/ecdsa", http.Header{"Authorization": []string{"Bearer " + ecdsaToken}}, "", http.StatusOK, "", ""},
		{"ECDSA JWKS with crv/x/y: kid=es256-crv-x-y", "/jwks/ecdsa", http.Header{"Authorization": []string{"Bearer " + ecdsaToken2}}, "", http.StatusOK, "", ""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, strings.NewReader(tc.body))
			helper.Must(err)

			req.Header = tc.header

			res, err := client.Do(req)
			helper.Must(err)

			message := getFirstAccessLogMessage(hook)
			if res.StatusCode != tc.status {
				subT.Errorf("expected Status %d, got: %d (%s)", tc.status, res.StatusCode, message)
				return
			}

			if tc.wantErrLog == "" {
				if message != "" {
					subT.Errorf("Expected error log: %q, actual: %#v", tc.wantErrLog, message)
				}
			} else {
				if !strings.HasPrefix(message, tc.wantErrLog) {
					subT.Errorf("Expected error log message: '%s', actual: '%s'", tc.wantErrLog, message)
				}
			}

			if res.StatusCode != http.StatusOK {
				return
			}

			expSub := "1234567890"
			if sub := res.Header.Get("X-Jwt-Sub"); sub != expSub {
				subT.Errorf("expected sub: %q, actual: %q", expSub, sub)
				return
			}

			if grantedPermissions := res.Header.Get("X-Granted-Permissions"); grantedPermissions != tc.expPerm {
				subT.Errorf("expected granted permissions: %q, actual: %q", tc.expPerm, grantedPermissions)
				return
			}
		})
	}
}

func TestJWKsMaxStale(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	config := `
	  server {
	    endpoint "/" {
	    access_control = ["stale"]
	      response {
	        body = "hi"
	      }
	    }
	  }
	  definitions {
	    jwt "stale" {
	      jwks_url = "${env.COUPER_TEST_BACKEND_ADDR}/jwks.json"
	      jwks_ttl = "3s"
	      jwks_max_stale = "2s"
	      backend {
	        origin = env.COUPER_TEST_BACKEND_ADDR
	        set_request_headers = {
	          Self-Destruct: ` + fmt.Sprint(time.Now().Add(2*time.Second).Unix()) + `
	        }
	      }
	    }
	  }
	`

	shutdown, hook, err := newCouperWithBytes([]byte(config), helper)
	defer shutdown()
	helper.Must(err)

	req, err := http.NewRequest(http.MethodGet, "http://back.end:8080/", nil)
	helper.Must(err)

	rsaToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.AZ0gZVqPe9TjjjJO0GnlTvERBXhPyxW_gTn050rCoEkseFRlp4TYry7WTQ7J4HNrH3btfxaEQLtTv7KooVLXQyMDujQbKU6cyuYH6MZXaM0Co3Bhu0awoX-2GVk997-7kMZx2yvwIR5ypd1CERIbNs5QcQaI4sqx_8oGrjO5ZmOWRqSpi4Mb8gJEVVccxurPu65gPFq9esVWwTf4cMQ3GGzijatnGDbRWs_igVGf8IAfmiROSVd17fShQtfthOFd19TGUswVAleOftC7-DDeJgAK8Un5xOHGRjv3ypK_6ZLRonhswaGXxovE0kLq4ZSzumQY2hOFE6x_BbrR1WKtGw"

	req.Header = http.Header{"Authorization": []string{"Bearer " + rsaToken}}

	res, err := client.Do(req)
	helper.Must(err)
	if res.StatusCode != 200 {
		message := getFirstAccessLogMessage(hook)
		t.Fatalf("expected status %d, got: %d (%s)", 200, res.StatusCode, message)
	}

	time.Sleep(3 * time.Second)
	// TTL 3s, backend is already failing, responds with stale JWKS

	res, err = client.Do(req)
	helper.Must(err)
	if res.StatusCode != 200 {
		message := getFirstAccessLogMessage(hook)
		t.Fatalf("expected status %d, got: %d (%s)", 200, res.StatusCode, message)
	}

	time.Sleep(3 * time.Second)
	// stale time (2s) exhausted -> 403
	res, err = client.Do(req)
	helper.Must(err)

	time.Sleep(time.Second)
	message := getFirstAccessLogMessage(hook)
	if res.StatusCode != 403 {
		t.Fatalf("expected status %d, got: %d (%s)", 403, res.StatusCode, message)
	}

	expectedMessage := "access control error: stale: received no valid JWKs data: <nil>, status code 500"
	if message != expectedMessage {
		t.Fatalf("expected message %q, got: %q", expectedMessage, message)
	}
}

func TestJWTAccessControlSourceConfig(t *testing.T) {
	helper := test.New(t)
	couperConfig, err := configload.LoadFile("testdata/integration/config/05_couper.hcl", "")
	helper.Must(err)

	log, _ := logrustest.NewNullLogger()
	ctx := context.TODO()

	expectedMsg := "configuration error: invalid-source: token source is invalid"

	err = command.NewRun(ctx).Execute(nil, couperConfig, log.WithContext(ctx))
	logErr, _ := err.(errors.GoError)
	if logErr == nil {
		t.Error("logErr should not be nil")
	} else if logErr.LogError() != expectedMsg {
		t.Errorf("\nwant:\t%s\ngot:\t%v", expectedMsg, logErr.LogError())
	}
}

func TestJWTAccessControl_round(t *testing.T) {
	pid := "asdf"
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/08_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name      string
		path      string
		expGroups []interface{}
	}

	for _, tc := range []testCase{
		{"separate jwt_signing_profile/jwt", "/separate", []interface{}{"g1", "g2"}},
		{"self-signed jwt", "/self-signed", []interface{}{}},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://back.end:8080%s/%s/create-jwt", tc.path, pid), nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Fatalf("%q: token request: unexpected status: %d", tc.name, res.StatusCode)
			}

			token := res.Header.Get("X-Jwt")

			req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("http://back.end:8080%s/%s/jwt", tc.path, pid), nil)
			helper.Must(err)
			req.Header.Set("Authorization", "Bearer "+token)

			res, err = client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Fatalf("%q: resource request: unexpected status: %d", tc.name, res.StatusCode)
			}

			decoder := json.NewDecoder(res.Body)
			var claims map[string]interface{}
			err = decoder.Decode(&claims)
			helper.Must(err)

			if _, ok := claims["exp"]; !ok {
				subT.Fatalf("%q: missing exp claim: %#v", tc.name, claims)
			}
			issclaim, ok := claims["iss"]
			if !ok {
				subT.Fatalf("%q: missing iss claim: %#v", tc.name, claims)
			}
			if issclaim != "the_issuer" {
				subT.Fatalf("%q: unexpected iss claim: %q", tc.name, issclaim)
			}
			pidclaim, ok := claims["pid"]
			if !ok {
				subT.Fatalf("%q: missing pid claim: %#v", tc.name, claims)
			}
			if pidclaim != pid {
				subT.Fatalf("%q: unexpected pid claim: %q", tc.name, pidclaim)
			}
			groupsclaim, ok := claims["groups"]
			if !ok {
				subT.Fatalf("%q: missing groups claim: %#v", tc.name, claims)
			}
			groupsclaimArray, ok := groupsclaim.([]interface{})
			if !ok {
				subT.Fatalf("%q: groups must be array: %#v", tc.name, groupsclaim)
			}
			if !cmp.Equal(tc.expGroups, groupsclaimArray) {
				subT.Errorf(cmp.Diff(tc.expGroups, groupsclaimArray))
			}
		})
	}
}

func TestJWT_CacheControl_private(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.qSLnmYgnkcOjxlOjFhUHQpCfTQ5elzKY3Mq6gRVT4iI"
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/10_couper.hcl", test.New(t))
	defer shutdown()

	var noCC []string

	type testCase struct {
		name      string
		path      string
		setToken  bool
		expStatus int
		expCC     []string
	}

	for _, tc := range []testCase{
		{"no token; no cc from ep", "/cc-private/no-cc", false, 401, []string{"private"}},
		{"no token; cc public from ep", "/cc-private/cc-public", false, 401, []string{"private"}},
		{"no token; no cc from ep; disable", "/no-cc-private/no-cc", false, 401, noCC},
		{"no token; cc public from ep; disable", "/no-cc-private/cc-public", false, 401, noCC},
		{"token; no cc from ep", "/cc-private/no-cc", true, 204, []string{"private"}},
		{"token; cc public from ep", "/cc-private/cc-public", true, 204, []string{"private", "public"}},
		{"token; no public cc from ep; disable", "/no-cc-private/no-cc", true, 204, noCC},
		{"token; cc public from ep; disable", "/no-cc-private/cc-public", true, 204, []string{"public"}},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)
			if tc.setToken {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.expStatus {
				subT.Errorf("expected Status %d, got: %d", tc.expStatus, res.StatusCode)
				return
			}

			cc := res.Header.Values("Cache-Control")
			sort.Strings(cc)

			if !cmp.Equal(tc.expCC, cc) {
				subT.Errorf("%s", cmp.Diff(tc.expCC, cc))
			}
		})
	}
}

func getFirstAccessLogMessage(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_access" && entry.Message != "" {
			return entry.Message
		}
	}

	return ""
}

func Test_Permissions(t *testing.T) {
	h := test.New(t)
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/09_couper.hcl", test.New(t))
	defer shutdown()

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"scp": "a",
		"rl":  "r1",
	})
	token, tokenErr := tok.SignedString([]byte("asdf"))
	h.Must(tokenErr)

	type testCase struct {
		name         string
		method       string
		path         string
		authorize    bool
		status       int
		wantGranted  string
		wantRequired string
		wantErrLog   string
		wantErrType  string
	}

	for _, tc := range []testCase{
		{"by scope: unauthorized", http.MethodGet, "/scope/foo", false, http.StatusUnauthorized, ``, ``, "access control error: scoped_jwt: token required", "jwt_token_missing"},
		{"by scope: no permission required by endpoint", http.MethodGet, "/scope/foo", true, http.StatusNoContent, `["a"]`, ``, "", ""},
		{"by scope: permission required by endpoint: insufficient permissions", http.MethodPost, "/scope/foo", true, http.StatusForbidden, ``, ``, `access control error: required permission "foo" not granted`, "beta_insufficient_permissions"},
		{"by scope: method not permitted", http.MethodDelete, "/scope/foo", true, http.StatusMethodNotAllowed, ``, ``, "method not allowed error: method DELETE not allowed by beta_required_permission", ""},
		{"by scope: permission required by endpoint via *: insufficient permissions", http.MethodGet, "/scope/bar", true, http.StatusForbidden, ``, `more`, `access control error: required permission "more" not granted`, "beta_insufficient_permissions"},
		{"by scope: no permission required by endpoint", http.MethodDelete, "/scope/bar", true, http.StatusNoContent, `["a"]`, ``, "", ""},
		{"by scope: required permission expression", http.MethodGet, "/scope/path/a/path", true, http.StatusNoContent, `["a"]`, ``, "", ""},
		{"by scope: required permission object expression (GET)", http.MethodGet, "/scope/object/get", true, http.StatusNoContent, `["a"]`, ``, "", ""},
		{"by scope: required permission object expression (DELETE)", http.MethodDelete, "/scope/object/delete", true, http.StatusForbidden, ``, ``, `access control error: required permission "z" not granted`, "beta_insufficient_permissions"},
		{"by scope: required permission bad expression", http.MethodGet, "/scope/bad/expression", true, http.StatusInternalServerError, ``, ``, "expression evaluation error", "evaluation"},
		{"by scope: required permission bad type number", http.MethodGet, "/scope/bad/type/number", true, http.StatusInternalServerError, ``, ``, "expression evaluation error", "evaluation"},
		{"by scope: required permission bad type boolean", http.MethodGet, "/scope/bad/type/boolean", true, http.StatusInternalServerError, ``, ``, "expression evaluation error", "evaluation"},
		{"by scope: required permission bad type tuple", http.MethodGet, "/scope/bad/type/tuple", true, http.StatusInternalServerError, ``, ``, "expression evaluation error", "evaluation"},
		{"by scope: required permission bad type null", http.MethodGet, "/scope/bad/type/null", true, http.StatusInternalServerError, ``, ``, "expression evaluation error", "evaluation"},
		{"by scope: required permission by api only: insufficient permissions", http.MethodGet, "/scope/permission-from-api", true, http.StatusForbidden, ``, ``, `access control error: required permission "z" not granted`, "beta_insufficient_permissions"},
		{"by role: unauthorized", http.MethodGet, "/role/foo", false, http.StatusUnauthorized, ``, ``, "access control error: roled_jwt: token required", "jwt_token_missing"},
		{"by role: sufficient permission", http.MethodGet, "/role/foo", true, http.StatusNoContent, `["a","b"]`, ``, "", ""},
		{"by role: permission required by endpoint: insufficient permissions", http.MethodPost, "/role/foo", true, http.StatusForbidden, ``, ``, `access control error: required permission "foo" not granted`, "beta_insufficient_permissions"},
		{"by role: method not permitted", http.MethodDelete, "/role/foo", true, http.StatusMethodNotAllowed, ``, ``, "method not allowed error: method DELETE not allowed by beta_required_permission", ""},
		{"by role: permission required by endpoint via *: insufficient permissions", http.MethodGet, "/role/bar", true, http.StatusForbidden, ``, `more`, `access control error: required permission "more" not granted`, "beta_insufficient_permissions"},
		{"by role: no permission required by endpoint", http.MethodDelete, "/role/bar", true, http.StatusNoContent, `["a","b"]`, ``, "", ""},
		{"by scope/role, mapped from scope", http.MethodGet, "/scope_and_role/foo", true, http.StatusNoContent, `["a","b","c","d","e"]`, ``, "", ""},
		{"by scope/role, mapped scope mapped from role", http.MethodGet, "/scope_and_role/bar", true, http.StatusNoContent, `["a","b","c","d","e"]`, ``, "", ""},
		{"by scope/role, mapped from scope, map files", http.MethodGet, "/scope_and_role_files/foo", true, http.StatusNoContent, `["a","b","c","d","e"]`, ``, "", ""},
		{"by scope/role, mapped scope mapped from role, map files", http.MethodGet, "/scope_and_role_files/bar", true, http.StatusNoContent, `["a","b","c","d","e"]`, ``, "", ""},
	} {
		t.Run(fmt.Sprintf("%s_%s_%s", tc.name, tc.method, tc.path), func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(tc.method, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			if tc.authorize {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				subT.Fatalf("expected Status %d, got: %d", tc.status, res.StatusCode)
			}

			granted := res.Header.Get("x-granted-permissions")
			if granted != tc.wantGranted {
				subT.Errorf("Expected granted permissions:\nWant:\t%q\nGot:\t%q", tc.wantGranted, granted)
			}

			required := res.Header.Get("x-required-permission")
			if required != tc.wantRequired {
				subT.Errorf("Expected required permission:\nWant:\t%q\nGot:\t%q", tc.wantRequired, required)
			}

			message := getFirstAccessLogMessage(hook)
			if !strings.HasPrefix(message, tc.wantErrLog) {
				subT.Errorf("Expected error log:\nWant:\t%q\nGot:\t%q", tc.wantErrLog, message)
			}

			errorType := getAccessLogErrorType(hook)
			if errorType != tc.wantErrType {
				subT.Errorf("Expected error type: %q, actual: %q", tc.wantErrType, errorType)
			}
		})
	}
}

func getAccessLogURL(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_access" && entry.Data["url"] != "" {
			if u, ok := entry.Data["url"].(string); ok {
				return u
			}
		}
	}

	return ""
}

func getAccessLogErrorType(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_access" && entry.Data["error_type"] != "" {
			if errorType, ok := entry.Data["error_type"].(string); ok {
				return errorType
			}
		}
	}

	return ""
}

func TestWrapperHiJack_WebsocketUpgrade(t *testing.T) {
	helper := test.New(t)
	shutdown, _ := newCouper("testdata/integration/api/04_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://connect.ws:8080/upgrade", nil)
	helper.Must(err)
	req.Close = false

	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")

	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	helper.Must(err)
	defer conn.Close()

	helper.Must(req.Write(conn))

	helper.Must(conn.SetDeadline(time.Time{}))

	textConn := textproto.NewConn(conn)
	_, _, _ = textConn.ReadResponse(http.StatusSwitchingProtocols) // ignore short resp error
	header, err := textConn.ReadMIMEHeader()
	helper.Must(err)

	expectedHeader := textproto.MIMEHeader{
		"Abc":               []string{"123"},
		"Connection":        []string{"Upgrade"},
		"Couper-Request-Id": header.Values("Couper-Request-Id"), // dynamic
		"Server":            []string{"couper.io"},
		"Upgrade":           []string{"websocket"},
	}

	if !reflect.DeepEqual(expectedHeader, header) {
		t.Errorf("Want: %v, got: %v", expectedHeader, header)
	}

	n, err := conn.Write([]byte("ping"))
	helper.Must(err)

	if n != 4 {
		t.Errorf("Expected 4 written bytes for 'ping', got: %d", n)
	}

	p := make([]byte, 4)
	_, err = conn.Read(p)
	helper.Must(err)

	if !bytes.Equal(p, []byte("pong")) {
		t.Errorf("Expected pong answer, got: %q", string(p))
	}
}

func TestWrapperHiJack_WebsocketUpgradeModifier(t *testing.T) {
	helper := test.New(t)
	shutdown, _ := newCouper("testdata/integration/api/13_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://connect.ws:8080/upgrade/ws", bytes.NewBufferString("ws-client-body"))
	helper.Must(err)
	req.Close = false

	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")

	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	helper.Must(err)
	defer conn.Close()

	helper.Must(req.Write(conn))

	helper.Must(conn.SetDeadline(time.Time{}))

	textConn := textproto.NewConn(conn)
	_, _, _ = textConn.ReadResponse(http.StatusSwitchingProtocols) // ignore short resp error
	header, err := textConn.ReadMIMEHeader()
	helper.Must(err)

	expectedHeader := textproto.MIMEHeader{
		"Abc":               {"123"},
		"Connection":        {"Upgrade"},
		"Couper-Request-Id": header.Values("Couper-Request-Id"), // dynamic
		"Echo":              {"ECHO"},
		"Server":            {"couper.io"},
		"Upgrade":           {"websocket"},
		"X-Body":            {"ws-client-body"},
		"X-Upgrade-Body":    {"ws-client-body"},
	}

	if !reflect.DeepEqual(expectedHeader, header) {
		t.Errorf(cmp.Diff(expectedHeader, header))
	}

	n, err := conn.Write([]byte("ping"))
	helper.Must(err)

	if n != 4 {
		t.Errorf("Expected 4 written bytes for 'ping', got: %d", n)
	}

	p := make([]byte, 4)
	_, err = conn.Read(p)
	helper.Must(err)

	if !bytes.Equal(p, []byte("pong")) {
		t.Errorf("Expected pong answer, got: %q", string(p))
	}
}

func TestWrapperHiJack_WebsocketUpgradeBodyBuffer(t *testing.T) {
	helper := test.New(t)
	shutdown, _ := newCouper("testdata/integration/api/13_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://connect.ws:8080/upgrade/small", bytes.NewBufferString("client-body"))
	helper.Must(err)
	req.Close = false

	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	helper.Must(err)
	defer conn.Close()

	helper.Must(req.Write(conn))

	helper.Must(conn.SetDeadline(time.Time{}))

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusOK, got: %s", res.Status)
	}

	expectedHeader := http.Header{
		"Content-Length":    res.Header.Values("Content-Length"), // dynamic, could change for other tests, unrelated here
		"Content-Type":      {"text/plain; charset=utf-8"},
		"Couper-Request-Id": res.Header.Values("Couper-Request-Id"), // dynamic
		"Date":              res.Header.Values("Date"),              // dynamic
		"Server":            {"couper.io"},
		"X-Body":            {"client-body"},
		"X-Resp-Body":       {"1234567890"},
	}

	if !reflect.DeepEqual(expectedHeader, res.Header) {
		t.Errorf(cmp.Diff(expectedHeader, res.Header))
	}
}

func TestWrapperHiJack_WebsocketUpgradeTimeout(t *testing.T) {
	helper := test.New(t)
	shutdown, _ := newCouper("testdata/integration/api/14_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://connect.ws:8080/upgrade", nil)
	helper.Must(err)
	req.Close = false

	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")

	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	helper.Must(err)
	defer conn.Close()

	helper.Must(req.Write(conn))

	helper.Must(conn.SetDeadline(time.Time{}))

	p := make([]byte, 77)
	_, err = conn.Read(p)
	helper.Must(err)

	if !bytes.HasPrefix(p, []byte("HTTP/1.1 504 Gateway Timeout\r\n")) {
		t.Errorf("Expected 504 status and related headers, got:\n%q", string(p))
	}
}

func TestAccessControl_Files_SPA(t *testing.T) {
	shutdown, _ := newCouper("testdata/file_serving/conf_ac.hcl", test.New(t))
	defer shutdown()

	client := newClient()

	type testCase struct {
		path      string
		password  string
		expStatus int
	}

	for _, tc := range []testCase{
		{"/favicon.ico", "", http.StatusUnauthorized},
		{"/robots.txt", "", http.StatusUnauthorized},
		{"/app", "", http.StatusUnauthorized},
		{"/app/1", "", http.StatusUnauthorized},
		{"/favicon.ico", "hans", http.StatusNotFound},
		{"/robots.txt", "hans", http.StatusOK},
		{"/app", "hans", http.StatusOK},
		{"/app/1", "hans", http.StatusOK},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://protect.me:8080"+tc.path, nil)
			helper.Must(err)

			if tc.password != "" {
				req.SetBasicAuth("", tc.password)
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.expStatus {
				subT.Errorf("Expected status: %d, got: %d", tc.expStatus, res.StatusCode)
			}
		})
	}
}

func TestHTTPServer_MultiAPI(t *testing.T) {
	client := newClient()

	type expectation struct {
		Path string
	}

	type testCase struct {
		path string
		exp  expectation
	}

	shutdown, _ := newCouper("testdata/integration/api/05_couper.hcl", test.New(t))
	defer shutdown()

	for _, tc := range []testCase{
		{"/v1/xxx", expectation{
			Path: "/v1/xxx",
		}},
		{"/v2/yyy", expectation{
			Path: "/v2/yyy",
		}},
		{"/v3/zzz", expectation{
			Path: "/v3/zzz",
		}},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestFunctions(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/functions/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name   string
		path   string
		header map[string]string
		status int
	}

	for _, tc := range []testCase{
		{"merge", "/v1/merge", map[string]string{"X-Merged-1": "{\"foo\":[1,2]}", "X-Merged-2": "{\"bar\":[3,4]}", "X-Merged-3": "[\"a\",\"b\"]"}, http.StatusOK},
		{"coalesce", "/v1/coalesce?q=a", map[string]string{"X-Coalesce-1": "/v1/coalesce", "X-Coalesce-2": "default", "X-Coalesce-3": "default", "X-Coalesce-4": "default"}, http.StatusOK},
		{"default", "/v1/default?q=a", map[string]string{
			"X-Default-1":  "/v1/default",
			"X-Default-2":  "default",
			"X-Default-3":  "default",
			"X-Default-4":  "default",
			"X-Default-5":  "prefix-default",
			"X-Default-6":  "default",
			"X-Default-7":  "default",
			"X-Default-8":  "default-8",
			"X-Default-9":  "",
			"X-Default-10": "",
			"X-Default-11": "0",
			"X-Default-12": "",
			"X-Default-13": `{"a":1}`,
			"X-Default-14": `{"a":1}`,
			"X-Default-15": `[1,2]`,
		}, http.StatusOK},
		{"contains", "/v1/contains", map[string]string{
			"X-Contains-1":  "yes",
			"X-Contains-2":  "no",
			"X-Contains-3":  "yes",
			"X-Contains-4":  "no",
			"X-Contains-5":  "yes",
			"X-Contains-6":  "no",
			"X-Contains-7":  "yes",
			"X-Contains-8":  "no",
			"X-Contains-9":  "yes",
			"X-Contains-10": "no",
			"X-Contains-11": "yes",
		}, http.StatusOK},
		{"length", "/v1/length", map[string]string{
			"X-Length-1": "2",
			"X-Length-2": "0",
			"X-Length-3": "5",
			"X-Length-4": "2",
		}, http.StatusOK},
		{"join", "/v1/join", map[string]string{
			"X-Join-1": "0-1-a-b-3-c-1.234-true-false",
			"X-Join-2": "||",
			"X-Join-3": "0-1-2-3-4",
		}, http.StatusOK},
		{"keys", "/v1/keys", map[string]string{
			"X-Keys-1": `["a","b","c"]`,
			"X-Keys-2": `[]`,
			"X-Keys-3": `["couper-request-id","user-agent"]`,
		}, http.StatusOK},
		{"set_intersection", "/v1/set_intersection", map[string]string{
			"X-Set_Intersection-1":  `[1,3]`,
			"X-Set_Intersection-2":  `[1,3]`,
			"X-Set_Intersection-3":  `[1,3]`,
			"X-Set_Intersection-4":  `[1,3]`,
			"X-Set_Intersection-5":  `[3]`,
			"X-Set_Intersection-6":  `[3]`,
			"X-Set_Intersection-7":  `[]`,
			"X-Set_Intersection-8":  `[]`,
			"X-Set_Intersection-9":  `[]`,
			"X-Set_Intersection-10": `[]`,
			"X-Set_Intersection-11": `[2.2]`,
			"X-Set_Intersection-12": `["b","d"]`,
			"X-Set_Intersection-13": `[true]`,
			"X-Set_Intersection-14": `[{"a":1}]`,
			"X-Set_Intersection-15": `[[1,2]]`,
		}, http.StatusOK},
		{"lookup", "/v1/lookup", map[string]string{
			"X-Lookup-1": "1",
			"X-Lookup-2": "default",
			"X-Lookup-3": "Go-http-client/1.1",
			"X-Lookup-4": "default",
		}, http.StatusOK},
		{"trim", "/v1/trim", map[string]string{
			"X-Trim": "foo \tbar",
		}, http.StatusOK},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
			}

			for k, v := range tc.header {
				if v1 := res.Header.Get(k); v1 != v {
					subT.Fatalf("%q: unexpected header value for %q: got: %q, want: %q", tc.name, k, v1, v)
				}
			}
		})
	}
}

func TestFunction_to_number(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/functions/01_couper.hcl", test.New(t))
	defer shutdown()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1/to_number", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected Status %d, got: %d", http.StatusOK, res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	helper.Must(res.Body.Close())

	exp := `{"float-2_34":2.34,"float-_3":0.3,"from-env":3.14159,"int":34,"int-3_":3,"int-3_0":3,"null":null}`
	if string(resBytes) != exp {
		t.Fatalf("Unexpected result\nwant: %s\n got:  %s", exp, string(resBytes))
	}
}

func TestFunction_to_number_errors(t *testing.T) {
	client := newClient()

	shutdown, logHook := newCouper("testdata/integration/functions/01_couper.hcl", test.New(t))
	defer shutdown()

	wd, werr := os.Getwd()
	if werr != nil {
		t.Fatal(werr)
	}
	wd = wd + "/testdata/integration/functions"

	type testCase struct {
		name   string
		path   string
		expMsg string
	}

	for _, tc := range []testCase{
		{"string", "/v1/to_number/string", wd + `/01_couper.hcl:65,23-28: Invalid function argument; Invalid value for "v" parameter: cannot convert "two" to number; given string must be a decimal representation of a number.`},
		{"bool", "/v1/to_number/bool", wd + `/01_couper.hcl:73,23-27: Invalid function argument; Invalid value for "v" parameter: cannot convert bool to number.`},
		{"tuple", "/v1/to_number/tuple", wd + `/01_couper.hcl:81,23-24: Invalid function argument; Invalid value for "v" parameter: cannot convert tuple to number.`},
		{"object", "/v1/to_number/object", wd + `/01_couper.hcl:89,23-24: Invalid function argument; Invalid value for "v" parameter: cannot convert object to number.`},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusInternalServerError {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.name, http.StatusInternalServerError, res.StatusCode)
			}
			msg := logHook.LastEntry().Message
			if msg != tc.expMsg {
				subT.Fatalf("%q: expected log message\nwant: %q\ngot:  %q", tc.name, tc.expMsg, msg)
			}
		})
	}
}

func TestFunction_length_errors(t *testing.T) {
	client := newClient()

	shutdown, logHook := newCouper("testdata/integration/functions/01_couper.hcl", test.New(t))
	defer shutdown()

	wd, werr := os.Getwd()
	if werr != nil {
		t.Fatal(werr)
	}
	wd = wd + "/testdata/integration/functions"

	type testCase struct {
		name   string
		path   string
		expMsg string
	}

	for _, tc := range []testCase{
		{"object", "/v1/length/object", wd + `/01_couper.hcl:126,19-26: Error in function call; Call to function "length" failed: collection must be a list, a map or a tuple.`},
		{"string", "/v1/length/string", wd + `/01_couper.hcl:134,19-26: Error in function call; Call to function "length" failed: collection must be a list, a map or a tuple.`},
		{"null", "/v1/length/null", wd + `/01_couper.hcl:142,26-30: Invalid function argument; Invalid value for "collection" parameter: argument must not be null.`},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusInternalServerError {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.name, http.StatusInternalServerError, res.StatusCode)
			}
			msg := logHook.LastEntry().Message
			if msg != tc.expMsg {
				subT.Fatalf("%q: expected log message\nwant: %q\ngot:  %q", tc.name, tc.expMsg, msg)
			}
		})
	}
}

func TestFunction_lookup_errors(t *testing.T) {
	client := newClient()

	shutdown, logHook := newCouper("testdata/integration/functions/01_couper.hcl", test.New(t))
	defer shutdown()

	wd, werr := os.Getwd()
	if werr != nil {
		t.Fatal(werr)
	}
	wd = wd + "/testdata/integration/functions"

	type testCase struct {
		name   string
		path   string
		expMsg string
	}

	for _, tc := range []testCase{
		{"null inputMap", "/v1/lookup/inputMap-null", wd + `/01_couper.hcl:203,26-30: Invalid function argument; Invalid value for "inputMap" parameter: argument must not be null.`},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusInternalServerError {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.name, http.StatusInternalServerError, res.StatusCode)
			}
			msg := logHook.LastEntry().Message
			if msg != tc.expMsg {
				subT.Fatalf("%q: expected log message\nwant: %q\ngot:  %q", tc.name, tc.expMsg, msg)
			}
		})
	}
}

func TestEndpoint_Response(t *testing.T) {
	client := newClient()
	var redirSeen bool
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		redirSeen = true
		return fmt.Errorf("do not follow")
	}

	shutdown, logHook := newCouper("testdata/integration/endpoint_eval/17_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path          string
		expStatusCode int
	}

	for _, tc := range []testCase{
		{"/200", http.StatusOK},
		{"/200/this-is-my-resp-body", http.StatusOK},
		{"/204", http.StatusNoContent},
		{"/301", http.StatusMovedPermanently},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			if tc.expStatusCode == http.StatusMovedPermanently {
				if !redirSeen {
					subT.Errorf("expected a redirect response")
				}

				resp := logHook.LastEntry().Data["response"]
				fields := resp.(logging.Fields)
				headers := fields["headers"].(map[string]string)
				if headers["location"] != "https://couper.io/" {
					subT.Errorf("expected location header log")
				}
			} else {
				helper.Must(err)
			}

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			helper.Must(res.Body.Close())

			if res.StatusCode != tc.expStatusCode {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.path, tc.expStatusCode, res.StatusCode)
			}

			if logHook.LastEntry().Data["status"] != tc.expStatusCode {
				subT.Logf("%v", logHook.LastEntry())
				subT.Errorf("Expected statusCode log: %d", tc.expStatusCode)
			}

			if len(resBytes) > 0 {
				b, exist := logHook.LastEntry().Data["response"].(logging.Fields)["bytes"]
				if !exist || b != len(resBytes) {
					subT.Errorf("Want bytes log: %d\ngot:\t%v", len(resBytes), logHook.LastEntry())
				}
			}
		})
	}
}

func TestCORS_Configuration(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/config/06_couper.hcl", test.New(t))
	defer shutdown()

	requestMethod := "GET"
	requestHeaders := "Authorization"

	type testCase struct {
		path              string
		origin            string
		expAllowed        bool
		expAllowedMethods string
		expAllowedHeaders string
		expVaryPF         string
		expVary           string
		expVaryCred       string
	}

	for _, tc := range []testCase{
		{"/06_couper.hcl", "a.com", true, requestMethod, requestHeaders, "Origin,Access-Control-Request-Method,Access-Control-Request-Headers", "Origin,Accept-Encoding", "Origin,Accept-Encoding"},
		{"/spa/", "b.com", true, requestMethod, requestHeaders, "Origin,Access-Control-Request-Method,Access-Control-Request-Headers", "Origin,Accept-Encoding", "Origin,Accept-Encoding"},
		{"/api/", "c.com", true, requestMethod, requestHeaders, "Origin,Access-Control-Request-Method,Access-Control-Request-Headers", "Origin,Accept-Encoding", "Origin"},
		{"/06_couper.hcl", "no.com", false, "", "", "Origin", "Origin,Accept-Encoding", "Origin,Accept-Encoding"},
		{"/spa/", "", false, "", "", "Origin", "Origin,Accept-Encoding", "Origin,Accept-Encoding"},
		{"/api/", "no.com", false, "", "", "Origin", "Origin,Accept-Encoding", "Origin"},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			// preflight request
			req, err := http.NewRequest(http.MethodOptions, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			req.Header.Set("Access-Control-Request-Method", requestMethod)
			req.Header.Set("Access-Control-Request-Headers", requestHeaders)
			req.Header.Set("Origin", tc.origin)

			res, err := client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != http.StatusNoContent {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.path, http.StatusNoContent, res.StatusCode)
			}

			acao, acaoExists := res.Header["Access-Control-Allow-Origin"]
			acam, acamExists := res.Header["Access-Control-Allow-Methods"]
			acah, acahExists := res.Header["Access-Control-Allow-Headers"]
			acac, acacExists := res.Header["Access-Control-Allow-Credentials"]
			if tc.expAllowed {
				if !acaoExists || acao[0] != tc.origin {
					subT.Errorf("Expected allowed origin, got: %v", acao)
				}
				if !acamExists || acam[0] != tc.expAllowedMethods {
					subT.Errorf("Expected allowed methods, got: %v", acam)
				}
				if !acahExists || acah[0] != tc.expAllowedHeaders {
					subT.Errorf("Expected allowed headers, got: %v", acah)
				}
				if !acacExists || acac[0] != "true" {
					subT.Errorf("Expected allowed credentials, got: %v", acac)
				}
			} else {
				if acaoExists {
					subT.Errorf("Expected not allowed origin, got: %v", acao)
				}
				if acamExists {
					subT.Errorf("Expected not allowed methods, got: %v", acam)
				}
				if acahExists {
					subT.Errorf("Expected not allowed headers, got: %v", acah)
				}
				if acacExists {
					subT.Errorf("Expected not allowed credentials, got: %v", acac)
				}
			}
			vary, varyExists := res.Header["Vary"]
			if !varyExists || strings.Join(vary, ",") != tc.expVaryPF {
				subT.Errorf("Expected vary %q, got: %q", tc.expVaryPF, strings.Join(vary, ","))
			}

			// actual request lacking credentials -> rejected by basic_auth AC
			req, err = http.NewRequest(requestMethod, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			req.Header.Set("Origin", tc.origin)

			res, err = client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != http.StatusUnauthorized {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.path, http.StatusUnauthorized, res.StatusCode)
			}

			acao, acaoExists = res.Header["Access-Control-Allow-Origin"]
			acac, acacExists = res.Header["Access-Control-Allow-Credentials"]
			if tc.expAllowed {
				if !acaoExists || acao[0] != tc.origin {
					subT.Errorf("Expected allowed origin, got: %v", acao)
				}
				if !acacExists || acac[0] != "true" {
					subT.Errorf("Expected allowed credentials, got: %v", acac)
				}
			} else {
				if acaoExists {
					subT.Errorf("Expected not allowed origin, got: %v", acao)
				}
				if acacExists {
					subT.Errorf("Expected not allowed credentials, got: %v", acac)
				}
			}
			vary, varyExists = res.Header["Vary"]
			if !varyExists || strings.Join(vary, ",") != tc.expVary {
				subT.Errorf("Expected vary %q, got: %q", tc.expVary, strings.Join(vary, ","))
			}

			// actual request with credentials
			req, err = http.NewRequest(requestMethod, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Authorization", "Basic Zm9vOmFzZGY=")

			res, err = client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != http.StatusOK {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.path, http.StatusOK, res.StatusCode)
			}

			acao, acaoExists = res.Header["Access-Control-Allow-Origin"]
			acac, acacExists = res.Header["Access-Control-Allow-Credentials"]
			if tc.expAllowed {
				if !acaoExists || acao[0] != tc.origin {
					subT.Errorf("Expected allowed origin, got: %v", acao)
				}
				if !acacExists || acac[0] != "true" {
					subT.Errorf("Expected allowed credentials, got: %v", acac)
				}
			} else {
				if acaoExists {
					subT.Errorf("Expected not allowed origin, got: %v", acao)
				}
				if acacExists {
					subT.Errorf("Expected not allowed credentials, got: %v", acac)
				}
			}
			vary, varyExists = res.Header["Vary"]
			if !varyExists || strings.Join(vary, ",") != tc.expVaryCred {
				subT.Errorf("Expected vary %q, got: %q", tc.expVaryCred, strings.Join(vary, ","))
			}
		})
	}
}

func TestLog_Level(t *testing.T) {
	shutdown, hook := newCouper("testdata/integration/logs/03_couper.hcl", test.New(t))
	defer shutdown()

	client := newClient()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://my.upstream:8080/", nil)
	helper.Must(err)

	hook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}

	for _, entry := range hook.AllEntries() {
		if entry.Level != logrus.InfoLevel {
			t.Errorf("Expected info level, got: %v", entry.Level)
		}
	}
}

func TestOAuthPKCEFunctions(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/functions/02_couper.hcl", test.New(t))
	defer shutdown()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/pkce", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Fatalf("expected Status %d, got: %d", 200, res.StatusCode)
	}

	v1 := res.Header.Get("x-v-1")
	v2 := res.Header.Get("x-v-2")
	hv := res.Header.Get("x-hv")
	if v2 != v1 {
		t.Errorf("multiple calls to oauth2_verifier() must return the same value:\n\t%s\n\t%s", v1, v2)
	}
	s256 := oauth2.Base64urlSha256(v1)
	if hv != s256 {
		t.Errorf("call to internal_oauth_hashed_verifier() returns wrong value:\nactual:\t\t%s\nexpected:\t%s", hv, s256)
	}
	au, err := url.Parse(res.Header.Get("x-au-pkce"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("oauth2_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email" {
		t.Errorf("oauth2_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile email")
	}
	if auq.Get("code_challenge_method") != "S256" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "S256")
	}
	if auq.Get("code_challenge") != hv {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), hv)
	}
	if auq.Get("state") != "" {
		t.Errorf("oauth2_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != "" {
		t.Errorf("oauth2_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("oauth2_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
	au, err = url.Parse(res.Header.Get("x-au-pkce-rel"))
	helper.Must(err)
	auq = au.Query()
	if auq.Get("redirect_uri") != "http://example.com:8080/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://example.com:8080/oidc/callback")
	}

	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/pkce", nil)
	helper.Must(err)

	res, err = client.Do(req)
	helper.Must(err)

	cv1_n := res.Header.Get("x-v-1")
	if cv1_n == v1 {
		t.Errorf("calls to oauth2_verifier() on different requests must not return the same value:\n\t%s\n\t%s", v1, cv1_n)
	}
}

func TestOAuthStateFunctions(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/functions/02_couper.hcl", test.New(t))
	defer shutdown()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/csrf", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Fatalf("expected Status %d, got: %d", 200, res.StatusCode)
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-state"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("oauth2_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile" {
		t.Errorf("oauth2_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile")
	}
	if auq.Get("code_challenge_method") != "" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "")
	}
	if auq.Get("code_challenge") != "" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), "")
	}
	if auq.Get("state") != hv {
		t.Errorf("oauth2_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), hv)
	}
	if auq.Get("nonce") != "" {
		t.Errorf("oauth2_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("oauth2_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
}

func TestOIDCPKCEFunctions(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			rw.Header().Set("Content-Type", "Application/json")
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)
			return
		} else if req.URL.Path == "/jwks" {
			rw.Header().Set("Content-Type", "Application/json")
			_, werr := rw.Write([]byte(`{}`))
			helper.Must(werr)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _, err := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	helper.Must(err)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/pkce", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Fatalf("expected Status %d, got: %d", 200, res.StatusCode)
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-pkce"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("oauth2_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email" {
		t.Errorf("oauth2_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile email")
	}
	if auq.Get("code_challenge_method") != "S256" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "S256")
	}
	if auq.Get("code_challenge") != hv {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), hv)
	}
	if auq.Get("state") != "" {
		t.Errorf("oauth2_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != "" {
		t.Errorf("oauth2_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("oauth2_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
	au, err = url.Parse(res.Header.Get("x-au-pkce-rel"))
	helper.Must(err)
	auq = au.Query()
	if auq.Get("redirect_uri") != "http://example.com:8080/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://example.com:8080/oidc/callback")
	}
}

func TestOIDCNonceFunctions(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/jwks" {
			rw.Header().Set("Content-Type", "Application/json")
			_, werr := rw.Write([]byte(`{}`))
			helper.Must(werr)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _, err := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	helper.Must(err)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/csrf", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Fatalf("expected Status %d, got: %d", 200, res.StatusCode)
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-nonce"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("oauth2_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile" {
		t.Errorf("oauth2_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile")
	}
	if auq.Get("code_challenge_method") != "" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "")
	}
	if auq.Get("code_challenge") != "" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), "")
	}
	if auq.Get("state") != "" {
		t.Errorf("oauth2_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != hv {
		t.Errorf("oauth2_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), hv)
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("oauth2_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
}

func TestOIDCDefaultPKCEFunctions(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo",
			"code_challenge_methods_supported": ["S256"]
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)
			return
		} else if req.URL.Path == "/jwks" {
			rw.Header().Set("Content-Type", "Application/json")
			_, werr := rw.Write([]byte(`{}`))
			helper.Must(werr)
			return
		}

		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _, err := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	helper.Must(err)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/default", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Fatalf("expected Status %d, got: %d", 200, res.StatusCode)
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-default"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("oauth2_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email address" {
		t.Errorf("oauth2_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile email")
	}
	if auq.Get("code_challenge_method") != "S256" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "S256")
	}
	if auq.Get("code_challenge") != hv {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), hv)
	}
	if auq.Get("state") != "" {
		t.Errorf("oauth2_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != "" {
		t.Errorf("oauth2_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("oauth2_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
}

func TestOIDCDefaultNonceFunctions(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)
			return
		} else if req.URL.Path == "/jwks" {
			rw.Header().Set("Content-Type", "Application/json")
			_, werr := rw.Write([]byte(`{}`))
			helper.Must(werr)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _, err := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	helper.Must(err)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/default", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Fatalf("expected Status %d, got: %d", 200, res.StatusCode)
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-default"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("oauth2_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("oauth2_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email address" {
		t.Errorf("oauth2_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile")
	}
	if auq.Get("code_challenge_method") != "" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "")
	}
	if auq.Get("code_challenge") != "" {
		t.Errorf("oauth2_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), "")
	}
	if auq.Get("state") != "" {
		t.Errorf("oauth2_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != hv {
		t.Errorf("oauth2_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), hv)
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("oauth2_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
}

func TestAllowedMethods(t *testing.T) {
	client := newClient()

	confPath := "testdata/integration/config/11_couper.hcl"
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()

	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJzY29wZSI6ImZvbyJ9.7zkwmXTmzFTKHC0Qnpw7uQCcacogWUvi_JU56uWJlkw"

	type testCase struct {
		name           string
		method         string
		path           string
		requestHeaders http.Header
		status         int
		couperError    string
	}

	for _, tc := range []testCase{
		{"path not found, authorized", http.MethodGet, "/api1/not-found", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusNotFound, "route not found error"},

		{"unrestricted, authorized, GET", http.MethodGet, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"unrestricted, authorized, HEAD", http.MethodHead, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"unrestricted, authorized, POST", http.MethodPost, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"unrestricted, authorized, PUT", http.MethodPut, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"unrestricted, authorized, PATCH", http.MethodPatch, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"unrestricted, authorized, DELETE", http.MethodDelete, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"unrestricted, authorized, OPTIONS", http.MethodOptions, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"unrestricted, authorized, CONNECT", http.MethodConnect, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"unrestricted, authorized, TRACE", http.MethodTrace, "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"unrestricted, authorized, BREW", "BREW", "/api1/unrestricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"unrestricted, unauthorized, GET", http.MethodGet, "/api1/unrestricted", http.Header{}, http.StatusUnauthorized, "access control error"},
		{"unrestricted, unauthorized, BREW", "BREW", "/api1/unrestricted", http.Header{}, http.StatusUnauthorized, "access control error"},
		{"unrestricted, CORS preflight", http.MethodOptions, "/api1/unrestricted", http.Header{"Origin": []string{"https://www.example.com"}, "Access-Control-Request-Method": []string{"POST"}, "Access-Control-Request-Headers": []string{"Authorization"}}, http.StatusNoContent, ""},

		{"restricted, authorized, GET", http.MethodGet, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"restricted, authorized, HEAD", http.MethodHead, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, POST", http.MethodPost, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"restricted, authorized, PUT", http.MethodPut, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, PATCH", http.MethodPatch, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, DELETE", http.MethodDelete, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, OPTIONS", http.MethodOptions, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, CONNECT", http.MethodConnect, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, TRACE", http.MethodTrace, "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted, authorized, BREW", "BREW", "/api1/restricted", http.Header{"Authorization": []string{"Bearer " + token}}, http.StatusOK, ""},
		{"restricted, CORS preflight", http.MethodOptions, "/api1/restricted", http.Header{"Origin": []string{"https://www.example.com"}, "Access-Control-Request-Method": []string{"POST"}, "Access-Control-Request-Headers": []string{"Authorization"}}, http.StatusNoContent, ""},

		{"wildcard, GET", http.MethodGet, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, HEAD", http.MethodHead, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, POST", http.MethodPost, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, PUT", http.MethodPut, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, PATCH", http.MethodPatch, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, DELETE", http.MethodDelete, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, OPTIONS", http.MethodOptions, "/api1/wildcard", http.Header{}, http.StatusOK, ""},
		{"wildcard, CONNECT", http.MethodConnect, "/api1/wildcard", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"wildcard, TRACE", http.MethodTrace, "/api1/wildcard", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"wildcard, BREW", "BREW", "/api1/wildcard", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},

		{"wildcard and more, GET", http.MethodGet, "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, HEAD", http.MethodHead, "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, PoSt", "PoSt", "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, PUT", http.MethodPut, "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, PATCH", http.MethodPatch, "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, DELETE", http.MethodDelete, "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, OPTIONS", http.MethodOptions, "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},
		{"wildcard and more, CONNECT", http.MethodConnect, "/api1/wildcardAndMore", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"wildcard and more, TRACE", http.MethodTrace, "/api1/wildcardAndMore", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"wildcard and more, bReW", "bReW", "/api1/wildcardAndMore", http.Header{}, http.StatusOK, ""},

		{"blocked, GET", http.MethodGet, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, HEAD", http.MethodHead, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, POST", http.MethodPost, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, PUT", http.MethodPut, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, PATCH", http.MethodPatch, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, DELETE", http.MethodDelete, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, OPTIONS", http.MethodOptions, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, CONNECT", http.MethodConnect, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, TRACE", http.MethodTrace, "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"blocked, BREW", "BREW", "/api1/blocked", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},

		{"restricted methods override, GET", http.MethodGet, "/api2/restricted", http.Header{}, http.StatusOK, ""},
		{"restricted methods override, HEAD", http.MethodHead, "/api2/restricted", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted methods override, POST", http.MethodPost, "/api2/restricted", http.Header{}, http.StatusOK, ""},
		{"restricted methods override, PUT", http.MethodPut, "/api2/restricted", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted methods override, PATCH", http.MethodPatch, "/api2/restricted", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted methods override, DELETE", http.MethodDelete, "/api2/restricted", http.Header{}, http.StatusOK, ""},
		{"restricted methods override, OPTIONS", http.MethodOptions, "/api2/restricted", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted methods override, CONNECT", http.MethodConnect, "/api2/restricted", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted methods override, TRACE", http.MethodTrace, "/api2/restricted", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted methods override, BREW", "BREW", "/api2/restricted", http.Header{}, http.StatusOK, ""},

		{"restricted by api only, GET", http.MethodGet, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, HEAD", http.MethodHead, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, POST", http.MethodPost, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, PUT", http.MethodPut, "/api2/restrictedByApiOnly", http.Header{}, http.StatusOK, ""},
		{"restricted by api only, PATCH", http.MethodPatch, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, DELETE", http.MethodDelete, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, OPTIONS", http.MethodOptions, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, CONNECT", http.MethodConnect, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, TRACE", http.MethodTrace, "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"restricted by api only, BREW", "BREW", "/api2/restrictedByApiOnly", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},

		{"files, GET", http.MethodGet, "/index.html", http.Header{}, http.StatusOK, ""},
		{"files, HEAD", http.MethodHead, "/index.html", http.Header{}, http.StatusOK, ""},
		{"files, POST", http.MethodPost, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, PUT", http.MethodPut, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, PATCH", http.MethodPatch, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, DELETE", http.MethodDelete, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, OPTIONS", http.MethodOptions, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, CONNECT", http.MethodConnect, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, TRACE", http.MethodTrace, "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, BREW", "BREW", "/index.html", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"files, CORS preflight", http.MethodOptions, "/index.html", http.Header{"Origin": []string{"https://www.example.com"}, "Access-Control-Request-Method": []string{"POST"}, "Access-Control-Request-Headers": []string{"Authorization"}}, http.StatusNoContent, ""},

		{"spa, GET", http.MethodGet, "/app/foo", http.Header{}, http.StatusOK, ""},
		{"spa, HEAD", http.MethodHead, "/app/foo", http.Header{}, http.StatusOK, ""},
		{"spa, POST", http.MethodPost, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, PUT", http.MethodPut, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, PATCH", http.MethodPatch, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, DELETE", http.MethodDelete, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, OPTIONS", http.MethodOptions, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, CONNECT", http.MethodConnect, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, TRACE", http.MethodTrace, "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, BREW", "BREW", "/app/foo", http.Header{}, http.StatusMethodNotAllowed, "method not allowed error"},
		{"spa, CORS preflight", http.MethodOptions, "/app/foo", http.Header{"Origin": []string{"https://www.example.com"}, "Access-Control-Request-Method": []string{"POST"}, "Access-Control-Request-Headers": []string{"Authorization"}}, http.StatusNoContent, ""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			logHook.Reset()
			req, err := http.NewRequest(tc.method, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)
			req.Header = tc.requestHeaders

			res, err := client.Do(req)
			helper.Must(err)

			if tc.status != res.StatusCode {
				subT.Errorf("Unexpected status code given; want: %d; got: %d", tc.status, res.StatusCode)
			}

			couperError := res.Header.Get("Couper-Error")
			if tc.couperError != couperError {
				subT.Errorf("Unexpected couper-error given; want: %q; got: %q", tc.couperError, couperError)
			}
		})
	}
}

func TestAllowedMethodsCORS_Preflight(t *testing.T) {
	client := newClient()

	confPath := "testdata/integration/config/11_couper.hcl"
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()

	type testCase struct {
		name          string
		path          string
		requestMethod string
		status        int
		allowMethods  []string
		couperError   string
	}

	for _, tc := range []testCase{
		{"unrestricted, CORS preflight, POST allowed", "/api1/unrestricted", http.MethodPost, http.StatusNoContent, []string{"POST"}, ""},
		{"restricted, CORS preflight, POST allowed", "/api1/restricted", http.MethodPost, http.StatusNoContent, []string{"POST"}, ""}, // CORS preflight ok even if OPTIONS is otherwise not allowed
		{"restricted, CORS preflight, PUT not allowed", "/api1/restricted", http.MethodPut, http.StatusNoContent, nil, ""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			logHook.Reset()
			req, err := http.NewRequest(http.MethodOptions, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)
			req.Header.Set("Origin", "https://www.example.com")
			req.Header.Set("Access-Control-Request-Method", tc.requestMethod)

			res, err := client.Do(req)
			helper.Must(err)

			if tc.status != res.StatusCode {
				subT.Errorf("Unexpected status code given; want: %d; got: %d", tc.status, res.StatusCode)
			}

			allowMethods := res.Header.Values("Access-Control-Allow-Methods")
			if !cmp.Equal(tc.allowMethods, allowMethods) {
				subT.Errorf(cmp.Diff(tc.allowMethods, allowMethods))
			}

			couperError := res.Header.Get("Couper-Error")
			if tc.couperError != couperError {
				subT.Errorf("Unexpected couper-error given; want: %q; got: %q", tc.couperError, couperError)
			}
		})
	}
}

func TestEndpoint_ResponseNilEvaluation(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/endpoint_eval/20_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path      string
		expVal    bool
		expCtyVal string
	}

	for _, tc := range []testCase{
		{"/1stchild", true, ""},
		{"/2ndchild/no", false, ""},
		{"/child-chain/no", false, ""},
		{"/list-idx", true, ""},
		{"/list-idx-splat", true, ""},
		{"/list-idx/no", false, ""},
		{"/list-idx-chain/no", false, ""},
		{"/list-idx-key-chain/no", false, ""},
		{"/root/no", false, ""},
		{"/tpl", true, ""},
		{"/for", true, ""},
		{"/conditional/false", true, ""},
		{"/conditional/true", false, ""},
		{"/conditional/nested", true, ""},
		{"/conditional/nested/true", true, ""},
		{"/conditional/nested/false", true, ""},
		{"/functions/arg-items", true, `{"foo":"bar","obj":{"key":"val"},"xxxx":null}`},
		{"/functions/tuple-expr", true, `{"array":["a","b"]}`},
		{"/rte1", true, "2"},
		{"/rte2", true, "2"},
		{"/ie1", true, "2"},
		{"/ie2", true, "2"},
		{"/uoe1", true, "-2"},
		{"/uoe2", true, "true"},
		{"/bad/dereference/string?foo=bar", false, ""},
		{"/bad/dereference/array?foo=bar", false, ""},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			hook.Reset()
			defer func() {
				if subT.Failed() {
					time.Sleep(time.Millisecond * 100)
					for _, entry := range hook.AllEntries() {
						s, _ := entry.String()
						println(s)
					}
				}
			}()

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Errorf("Expected Status OK, got: %d", res.StatusCode)
				return
			}

			defer func() {
				if subT.Failed() {
					for k := range res.Header {
						subT.Logf("%s: %s", k, res.Header.Get(k))
					}
				}
			}()

			val, ok := res.Header[http.CanonicalHeaderKey("X-Value")]
			if !tc.expVal && ok {
				subT.Errorf("%q: expected no value, got: %q", tc.path, val)
			} else if tc.expVal && !ok {
				subT.Errorf("%q: expected X-Value header, got: nothing", tc.path)
			}

			if res.Header.Get("Z-Value") != "y" {
				subT.Errorf("additional header Z-Value should always been written")
			}

			if tc.expCtyVal != "" && tc.expCtyVal != val[0] {
				subT.Errorf("Want: %s, got: %v", tc.expCtyVal, val[0])
			}

		})
	}
}

func TestEndpoint_ConditionalEvaluationError(t *testing.T) {
	client := newClient()

	wd, werr := os.Getwd()
	if werr != nil {
		t.Fatal(werr)
	}
	wd = wd + "/testdata/integration/endpoint_eval"

	shutdown, hook := newCouper("testdata/integration/endpoint_eval/20_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path       string
		expMessage string
	}

	for _, tc := range []testCase{
		{"/conditional/null", wd + "/20_couper.hcl:281,16-20: Null condition; The condition value is null. Conditions must either be true or false."},
		{"/conditional/string", wd + "/20_couper.hcl:287,16-21: Incorrect condition type; The condition expression must be of type bool."},
		{"/conditional/number", wd + "/20_couper.hcl:293,16-17: Incorrect condition type; The condition expression must be of type bool."},
		{"/conditional/tuple", wd + "/20_couper.hcl:299,16-18: Incorrect condition type; The condition expression must be of type bool."},
		{"/conditional/object", wd + "/20_couper.hcl:305,16-18: Incorrect condition type; The condition expression must be of type bool."},
		{"/conditional/string/expr", wd + "/20_couper.hcl:311,16-30: Incorrect condition type; The condition expression must be of type bool."},
		{"/conditional/number/expr", wd + "/20_couper.hcl:317,16-26: Incorrect condition type; The condition expression must be of type bool."},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			hook.Reset()
			defer func() {
				if subT.Failed() {
					time.Sleep(time.Millisecond * 100)
					for _, entry := range hook.AllEntries() {
						s, _ := entry.String()
						println(s)
					}
				}
			}()

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusInternalServerError {
				subT.Errorf("Expected Status InternalServerError, got: %d", res.StatusCode)
				return
			}

			time.Sleep(time.Millisecond * 100)
			entry := hook.LastEntry()
			if entry != nil && entry.Level == logrus.ErrorLevel {
				if entry.Message != tc.expMessage {
					subT.Errorf("wrong error message,\nexp: %s\ngot: %s", tc.expMessage, entry.Message)
				}
			}
		})
	}
}

func TestEndpoint_ForLoop(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/endpoint_eval/21_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path      string
		header    http.Header
		expResult string
	}

	for _, tc := range []testCase{
		{"/for0", http.Header{}, `["a","b"]`},
		{"/for1", http.Header{}, `[0,1]`},
		{"/for2", http.Header{}, `{"a":0,"b":1}`},
		{"/for3", http.Header{}, `{"a":[0,1],"b":[2]}`},
		{"/for4", http.Header{}, `["a","b"]`},
		{"/for5", http.Header{"x-1": []string{"val1"}, "x-2": []string{"val2"}, "y": []string{`["x-1","x-2"]`}, "z": []string{"pfx"}}, `{"pfx-x-1":"val1","pfx-x-2":"val2"}`},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			req.Header = tc.header
			helper.Must(err)

			hook.Reset()

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != http.StatusOK {
				subT.Errorf("Expected Status OK, got: %d", res.StatusCode)
				return
			}

			result := string(resBytes)
			if result != tc.expResult {
				subT.Errorf("Want: %s, got: %v", tc.expResult, result)
			}
		})
	}
}

func TestWildcardURLAttribute(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/url/07_couper.hcl", test.New(t))
	defer shutdown()

	for _, testcase := range []struct{ path, expectedPath, expectedQuery string }{
		{"/req/anything", "/anything", ""},
		{"/req/anything/", "/anything/", ""},
		{"/req-query/anything/?a=b", "/anything/", "a=c"},
		{"/req-backend/anything/?a=b", "/anything/", "a=c"},
		{"/proxy/anything", "/anything", ""},
		{"/proxy/anything/", "/anything/", ""},
		{"/proxy-query/anything/?a=b", "/anything/", "a=c"},
		{"/proxy-backend/anything", "/anything", ""},
		{"/proxy-backend-rel/anything?a=b", "/anything", "a=c"},
		{"/proxy-backend-path/other-wildcard?a=b", "/anything", "a=c"},
	} {
		t.Run(testcase.path[1:], func(st *testing.T) {
			helper := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+testcase.path, nil)
			helper.Must(err)

			hook.Reset()

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				st.Error("expected status OK")
			}

			b, err := io.ReadAll(res.Body)
			helper.Must(res.Body.Close())
			helper.Must(err)

			type result struct {
				Path     string
				RawQuery string
			}
			r := result{}
			helper.Must(json.Unmarshal(b, &r))
			//st.Logf("%v", r)

			if testcase.expectedPath != r.Path {
				st.Errorf("Expected path: %q, got: %q", testcase.expectedPath, r.Path)
			}

			if testcase.expectedQuery != r.RawQuery {
				st.Errorf("Expected query: %q, got: %q", testcase.expectedQuery, r.RawQuery)
			}
		})
	}
}

func TestEnvironmentSetting(t *testing.T) {
	helper := test.New(t)
	tests := []struct {
		env string
	}{
		{"foo"},
		{"bar"},
	}

	template := `
	  server {
	    endpoint "/" {
	      response {
	        environment "foo" {
	          headers = { X-Env: "foo" }
	        }
	        environment "bar" {
	          headers = { X-Env: "bar" }
	        }
	      }
	    }
	  }
	  settings {
	    environment = "%s"
	  }
	`

	file, err := os.CreateTemp("", "tmpfile-")
	helper.Must(err)
	defer file.Close()
	defer os.Remove(file.Name())

	client := newClient()
	for _, tt := range tests {
		t.Run(tt.env, func(subT *testing.T) {
			config := []byte(fmt.Sprintf(template, tt.env))
			err := os.Truncate(file.Name(), 0)
			helper.Must(err)
			_, err = file.Seek(0, 0)
			helper.Must(err)
			_, err = file.Write(config)
			helper.Must(err)

			couperConfig, err := configload.LoadFile(file.Name(), "")
			helper.Must(err)

			shutdown, _ := newCouperWithConfig(couperConfig, helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if header := res.Header.Get("X-Env"); header != tt.env {
				subT.Errorf("Unexpected header:\n\tWant: %q\n\tGot:  %q", tt.env, header)
			}
		})
	}
}

func TestWildcardVsEmptyPathParams(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/url/08_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path     string
		expected string
	}

	for _, tc := range []testCase{
		{"/foo", "/**"},
		{"/p1/A/B/C", "/**"},
		{"/p1/A/B/", "/p1/{x}/{y}"},
		{"/p1/A//", "/**"},
		{"/p1///", "/**"},
		{"/p1//", "/**"},
		{"/p1/A/B", "/p1/{x}/{y}"},
		{"/p1/A/", "/**"},
		{"/p1/A", "/**"},
		{"/p1/", "/**"},
		{"/p1", "/**"},
		{"/p2/A/B/C", "/p2/**"},
		{"/p2/A/B/", "/p2/{x}/{y}"},
		{"/p2/A/B", "/p2/{x}/{y}"},
		{"/p2/A/", "/p2/**"},
		{"/p2/A", "/p2/**"},
		{"/p2/", "/p2/**"},
		{"/p2", "/p2/**"},
		{"/p3/A/B/C", "/p3/**"},
		{"/p3/A/B/", "/p3/{x}/{y}"},
		{"/p3/A/B", "/p3/{x}/{y}"},
		{"/p3/A/", "/p3/{x}"},
		{"/p3/A", "/p3/{x}"},
		{"/p3/", "/p3/**"},
		{"/p3", "/p3/**"},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Errorf("Unexpected status: want: 200, got %d", res.StatusCode)
			}

			match := res.Header.Get("Match")
			if match != tc.expected {
				subT.Errorf("Unexpected match for %s: want %s, got %s", tc.path, tc.expected, match)
			}
		})
	}
}
