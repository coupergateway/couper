package server_test

import (
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
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
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
	couperConfig, err := configload.LoadFile(filepath.Join(testWorkingDir, file))
	helper.Must(err)

	return newCouperWithConfig(couperConfig, helper)
}

// newCouperWithTemplate applies given variables first and loads Couper with the resulting configuration file.
// Example template:
// 		My {{.message}}
// Example value:
//		map[string]interface{}{
//			"message": "value",
//		}
func newCouperWithTemplate(file string, helper *test.Helper, vars map[string]interface{}) (func(), *logrustest.Hook) {
	if vars == nil {
		return newCouper(file, helper)
	}

	tpl, err := template.New(filepath.Base(file)).ParseFiles(file)
	helper.Must(err)

	result := &bytes.Buffer{}
	helper.Must(tpl.Execute(result, vars))

	return newCouperWithBytes(result.Bytes(), helper)
}

func newCouperWithBytes(file []byte, helper *test.Helper) (func(), *logrustest.Hook) {
	couperConfig, err := configload.LoadBytes(file, "couper-bytes.hcl")
	helper.Must(err)

	return newCouperWithConfig(couperConfig, helper)
}

func newCouperWithConfig(couperConfig *config.Couper, helper *test.Helper) (func(), *logrustest.Hook) {
	testServerMu.Lock()
	defer testServerMu.Unlock()

	log, hook := test.NewLogger()

	ctx, cancelFn := context.WithCancel(context.Background())
	shutdownFn := func() {
		if helper.TestFailed() { // log on error
			for _, entry := range hook.AllEntries() {
				helper.Logf(entry.String())
			}
		}
		cleanup(cancelFn, helper)
	}

	// ensure the previous test aren't listening
	port := couperConfig.Settings.DefaultPort
	round := time.Duration(0)
	for {
		round++
		conn, dialErr := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if dialErr != nil {
			break
		}
		_ = conn.Close()
		time.Sleep(time.Second + (time.Second*round)/2)

		if round == 10 {
			panic("port is still in use")
		}
	}

	go func() {
		if err := command.NewRun(ctx).Execute([]string{couperConfig.Filename}, couperConfig, log.WithContext(ctx)); err != nil {
			shutdownFn()
			panic(err)
		}
	}()

	time.Sleep(time.Second / 2)

	for _, entry := range hook.AllEntries() {
		if entry.Level < logrus.InfoLevel {
			helper.Must(fmt.Errorf("error: %#v: %s", entry.Data, entry.Message))
		}
	}

	hook.Reset() // no startup logs
	return shutdownFn, hook
}

func newClient() *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				_, port, _ := net.SplitHostPort(addr)
				if port != "" {
					return dialer.DialContext(ctx, "tcp4", "127.0.0.1:"+port)
				}
				return dialer.DialContext(ctx, "tcp4", "127.0.0.1")
			},
			DisableCompression: true,
		},
	}
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
				expectation{http.StatusOK, []byte(`<html><body><title>FS</title></body></html>`), nil, "file"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/foo"},
				expectation{http.StatusOK, []byte("<html><body><title>SPA_01</title></body></html>\n"), nil, "spa"},
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
					t.Errorf("Expected statusCode %d, got %d", rc.exp.status, res.StatusCode)
					subT.Logf("Failed: %s|%s", testcase.fileName, rc.req.url)
				}

				for k, v := range rc.exp.header {
					if !reflect.DeepEqual(res.Header[k], v) {
						t.Errorf("Exptected headers:\nWant:\t%#v\nGot:\t%#v\n", v, res.Header[k])
					}
				}

				if rc.exp.body != nil && !bytes.Equal(resBytes, rc.exp.body) {
					t.Errorf("Expected same body content:\nWant:\t%q\nGot:\t%q\n", string(rc.exp.body), string(resBytes))
				}

				entry := logHook.LastEntry()

				if entry == nil || entry.Data["type"] != "couper_access" {
					t.Error("Expected a log entry, got nothing")
					return
				}
				if handler, ok := entry.Data["handler"]; rc.exp.handlerName != "" && (!ok || handler != rc.exp.handlerName) {
					t.Errorf("Expected handler %q within logs, got:\n%#v", rc.exp.handlerName, entry.Data)
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

func TestHTTPServer_XFHHeader(t *testing.T) {
	client := newClient()

	env.OsEnviron = func() []string {
		return []string{"COUPER_XFH=true"}
	}
	defer func() { env.OsEnviron = os.Environ }()

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
					t.Errorf("Expected no header with key Content-Encoding, got value: %s", val)
				}
			} else {
				if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
					t.Errorf("Expected Content-Encoding header value: %q, got: %q", "gzip", ce)
				}

				body, err = gzip.NewReader(res.Body)
				helper.Must(err)
			}

			if vr := res.Header.Get("Vary"); vr != "Accept-Encoding" {
				t.Errorf("Expected Accept-Encoding header value %q, got: %q", "Vary", vr)
			}

			resBytes, err := io.ReadAll(body)
			helper.Must(err)

			srcBytes, err := os.ReadFile(filepath.Join(testWorkingDir, "testdata/integration/files/htdocs_c_gzip"+tc.path))
			helper.Must(err)

			if !bytes.Equal(resBytes, srcBytes) {
				t.Errorf("Want:\n%s\nGot:\n%s", string(srcBytes), string(resBytes))
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
				"ae_noop":     []string{"", ""},
				"ae_null":     []string{"", ""},
				"ae_string":   []string{"str", "str"},
				"ae":          []string{"ae", "ae"},
				"aeb_a_and_b": []string{"A&B", "A&B"},
				"aeb_empty":   []string{"", ""},
				"aeb_multi":   []string{"str1", "str2", "str3", "str4"},
				"aeb_noop":    []string{"", ""},
				"aeb_null":    []string{"", ""},
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
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
		{"/v1/uuu/foo", expectation{
			Path: "/xxx/xxx/api/foo",
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
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

	if p := hook.Entries[0].Data["request"].(logging.Fields)["path"]; p != "/path?query" {
		t.Errorf("Unexpected path given: %s", p)
	}
}

func TestHTTPServer_BackendLogPathInEndpoint(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/api/08_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/abc?query#fragment", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	if p := hook.Entries[0].Data["request"].(logging.Fields)["path"]; p != "/new/path/abc?query" {
		t.Errorf("Unexpected path given: %s", p)
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

	if m := hook.Entries[0].Message; m != "configuration error: path attribute: invalid fragment found in \"/path#xxx\"" {
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

	if m := hook.Entries[0].Message; m != "configuration error: path attribute: invalid query string found in \"/path?xxx\"" {
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

	if m := hook.Entries[0].Message; m != "configuration error: path_prefix attribute: invalid fragment found in \"/path#xxx\"" {
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

	if m := hook.Entries[0].Message; m != "configuration error: path_prefix attribute: invalid query string found in \"/path?xxx\"" {
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
				"Aeb_noop":    []string{"", ""},
				"Aeb_null":    []string{"", ""},
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
				t.Errorf("Missing or invalid header Remove-Me-1: %s", r1)
			}
			if r2 := res.Header.Get("Remove-Me-2"); r2 != "" {
				t.Errorf("Unexpected header %s", r2)
			}

			if s2 := res.Header.Get("Set-Me-2"); s2 != "s2" {
				t.Errorf("Missing or invalid header Set-Me-2: %s", s2)
			}

			if a2 := res.Header.Get("Add-Me-2"); a2 != "a2" {
				t.Errorf("Missing or invalid header Add-Me-2: %s", a2)
			}

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			jsonResult.Headers.Del("User-Agent")
			jsonResult.Headers.Del("X-Forwarded-For")
			jsonResult.Headers.Del("Couper-Request-Id")

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
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

	logHook.Reset()
	res, err := client.Do(req)
	helper.Must(err)

	if l := len(logHook.Entries); l != 2 {
		t.Fatalf("Unexpected number of log lines: %d", l)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	helper.Must(res.Body.Close())

	backendLog := logHook.Entries[0]
	accessLog := logHook.Entries[1]

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
				"x": []string{"y"},
			},
		}},
		{"04_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"x": []string{"y"},
			},
		}},
		{"05_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
				"x": []string{"y"},
			},
		}},
		{"06_couper.hcl", expectation{
			Path: "/anything",
			Query: url.Values{
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_DynamicRequest(t *testing.T) {
	client := newClient()

	configFile := "testdata/integration/endpoint_eval/13_couper.hcl"

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

			shutdown, _ := newCouper(configFile, helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080?method=put", nil)
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_request_bodies(t *testing.T) {
	client := newClient()

	configFile := "testdata/integration/endpoint_eval/14_couper.hcl"

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
				Body:   "\"foo\"",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"5"},
					"Content-Type":   []string{"application/json"},
				},
			},
		},
		{
			"/request/json_body/object",
			"",
			"",
			expectation{
				Body:   "{\"foo\":\"bar\"}",
				Args:   url.Values{},
				Method: "POST",
				Headers: http.Header{
					"Content-Length": []string{"13"},
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

			shutdown, _ := newCouper(configFile, helper)
			defer shutdown()

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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v", tc.exp, jsonResult)
			}
		})
	}
}

func TestHTTPServer_response_bodies(t *testing.T) {
	client := newClient()

	configFile := "testdata/integration/endpoint_eval/14_couper.hcl"

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
				Body:        "\"foo\"",
				ContentType: "application/json",
			},
		},
		{
			"/response/json_body/object",
			expectation{
				Body:        "{\"foo\":\"bar\"}",
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

			shutdown, _ := newCouper(configFile, helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			res.Body.Close()

			if string(resBytes) != tc.exp.Body {
				t.Errorf("%s: want: %s, got:%s", tc.path, tc.exp.Body, string(resBytes))
			}

			if ct := res.Header.Get("Content-Type"); ct != tc.exp.ContentType {
				t.Errorf("%s: want: %s, got:%s", tc.path, tc.exp.ContentType, ct)
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

	for _, tc := range []testCase{
		{"/my-waffik/my.host.de/" + testBackend.Addr()[7:], expectation{
			Host:   "my.host.de",
			Origin: testBackend.Addr()[7:],
			Path:   "/anything",
		}},
		{"/my-respo/my.host.com/" + testBackend.Addr()[7:], expectation{
			Host:   "my.host.com",
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			jsonResult.Origin = res.Header.Get("X-Origin")

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("want: %#v, got: %#v, payload:\n%s", tc.exp, jsonResult, string(resBytes))
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
		Url      string      `json:"url"`
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
		Url: "http://example.com:8080/req?foo=bar",
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
		Url      string                 `json:"url"`
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
		Url: "http://example.com:8080/req?foo=bar",
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
		Url      string      `json:"url"`
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
		Url: "http://example.com:8080/req?foo=bar",
	}

	if fmt.Sprint(jsonResult) != fmt.Sprint(exp) {
		t.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", exp, jsonResult, string(resBytes))
	}
}

func TestHTTPServer_AcceptingForwardedUrl(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/settings/05_couper.hcl")
	shutdown, hook := newCouper(confPath, test.New(t))
	defer shutdown()

	type expectation struct {
		Protocol string `json:"protocol"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Origin   string `json:"origin"`
		Url      string `json:"url"`
	}

	type testCase struct {
		name             string
		header           http.Header
		exp              expectation
		wantAccessLogUrl string
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
				Url:      "http://localhost:8080/path",
			},
			"http://localhost:8080/path",
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
				Url:      "https://www.example.com/path",
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
				Url:      "https://localhost:8443/path",
			},
			"https://localhost:8443/path",
		},
		{
			"host, port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com"},
				"X-Forwarded-Port": []string{"8443"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "http://www.example.com:8443",
				Url:      "http://www.example.com:8443/path",
			},
			"http://www.example.com:8443/path",
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
				Url:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.exp, jsonResult, string(resBytes))
			}

			url := getAccessLogUrl(hook)
			if url != tc.wantAccessLogUrl {
				t.Errorf("Expected URL: %q, actual: %q", tc.wantAccessLogUrl, url)
			}
		})
	}
}

func TestHTTPServer_XFH_AcceptingForwardedUrl(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/settings/06_couper.hcl")
	shutdown, hook := newCouper(confPath, test.New(t))
	defer shutdown()

	type expectation struct {
		Protocol string `json:"protocol"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Origin   string `json:"origin"`
		Url      string `json:"url"`
	}

	type testCase struct {
		name             string
		header           http.Header
		exp              expectation
		wantAccessLogUrl string
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
				Url:      "http://localhost:8080/path",
			},
			"http://localhost:8080/path",
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
				Url:      "https://www.example.com/path",
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
				Url:      "https://localhost:8443/path",
			},
			"https://localhost:8443/path",
		},
		{
			"host, port, no proto",
			http.Header{
				"X-Forwarded-Host": []string{"www.example.com"},
				"X-Forwarded-Port": []string{"8443"},
			},
			expectation{
				Protocol: "http",
				Host:     "www.example.com",
				Port:     8443,
				Origin:   "http://www.example.com:8443",
				Url:      "http://www.example.com:8443/path",
			},
			"http://www.example.com:8443/path",
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
				Url:      "https://www.example.com:8443/path",
			},
			"https://www.example.com:8443/path",
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.exp, jsonResult, string(resBytes))
			}

			url := getAccessLogUrl(hook)
			if url != tc.wantAccessLogUrl {
				t.Errorf("Expected URL: %q, actual: %q", tc.wantAccessLogUrl, url)
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
	shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"rsOrigin": ResourceOrigin.URL})
	defer shutdown()

	type expectation struct {
		Method   string                 `json:"method"`
		Protocol string                 `json:"protocol"`
		Host     string                 `json:"host"`
		Port     int64                  `json:"port"`
		Path     string                 `json:"path"`
		Query    map[string][]string    `json:"query"`
		Origin   string                 `json:"origin"`
		Url      string                 `json:"url"`
		Body     string                 `json:"body"`
		JsonBody map[string]interface{} `json:"json_body"`
		FormBody map[string][]string    `json:"form_body"`
	}

	type testCase struct {
		name   string
		relUrl string
		header http.Header
		body   io.Reader
		exp    expectation
	}

	helper := test.New(t)
	resourceOrigin, err := url.Parse(ResourceOrigin.URL)
	helper.Must(err)

	port, _ := strconv.ParseInt(resourceOrigin.Port(), 10, 64)

	for _, tc := range []testCase{
		{
			"body",
			"/body",
			http.Header{},
			strings.NewReader(`abcd1234`),
			expectation{
				Method:   "POST",
				Protocol: resourceOrigin.Scheme,
				Host:     resourceOrigin.Hostname(),
				Port:     port,
				Path:     "/resource",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   ResourceOrigin.URL,
				Url:      ResourceOrigin.URL + "/resource?foo=bar",
				Body:     "abcd1234",
				JsonBody: map[string]interface{}{},
				FormBody: map[string][]string{},
			},
		},
		{
			"json_body",
			"/json_body",
			http.Header{"Content-Type": []string{"application/json"}},
			strings.NewReader(`{"s":"abcd1234"}`),
			expectation{
				Method:   "POST",
				Protocol: resourceOrigin.Scheme,
				Host:     resourceOrigin.Hostname(),
				Port:     port,
				Path:     "/resource",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   ResourceOrigin.URL,
				Url:      ResourceOrigin.URL + "/resource?foo=bar",
				Body:     `{"s":"abcd1234"}`,
				JsonBody: map[string]interface{}{"s": "abcd1234"},
				FormBody: map[string][]string{},
			},
		},
		{
			"form_body",
			"/form_body",
			http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}},
			strings.NewReader(`s=abcd1234`),
			expectation{
				Method:   "POST",
				Protocol: resourceOrigin.Scheme,
				Host:     resourceOrigin.Hostname(),
				Port:     port,
				Path:     "/resource",
				Query:    map[string][]string{"foo": {"bar"}},
				Origin:   ResourceOrigin.URL,
				Url:      ResourceOrigin.URL + "/resource?foo=bar",
				Body:     `s=abcd1234`,
				JsonBody: map[string]interface{}{},
				FormBody: map[string][]string{"s": {"abcd1234"}},
			},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodPost, "http://localhost:8080"+tc.relUrl, tc.body)
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
				t.Errorf("%s: unmarshal json: %v: got:\n%s", tc.name, err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("%s\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.name, tc.exp, jsonResult, string(resBytes))
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
		Url      string                 `json:"url"`
		Body     string                 `json:"body"`
		JsonBody map[string]interface{} `json:"json_body"`
		FormBody map[string][]string    `json:"form_body"`
	}

	type testCase struct {
		name   string
		relUrl string
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
				Url:      "http://localhost:8080/body?foo=bar",
				Body:     "abcd1234",
				JsonBody: map[string]interface{}{},
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
				Url:      "http://localhost:8080/json_body?foo=bar",
				Body:     `{"s":"abcd1234"}`,
				JsonBody: map[string]interface{}{"s": "abcd1234"},
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
				Url:      "http://localhost:8080/form_body?foo=bar",
				Body:     `s=abcd1234`,
				JsonBody: map[string]interface{}{},
				FormBody: map[string][]string{"s": {"abcd1234"}},
			},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodPost, "http://localhost:8080"+tc.relUrl, tc.body)
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
				t.Errorf("%s: unmarshal json: %v: got:\n%s", tc.name, err, string(resBytes))
			}
			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("%s\nwant:\t%#v\ngot:\t%#v\npayload: %s", tc.name, tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_Endpoint_Evaluation_Inheritance(t *testing.T) {
	client := newClient()

	for _, confFile := range []string{"02_couper.hcl", "03_couper.hcl"} {
		confPath := path.Join("testdata/integration/endpoint_eval", confFile)

		type expectation struct {
			Path           string
			ResponseStatus int
		}

		type testCase struct {
			reqPath string
			exp     expectation
		}

		for _, tc := range []testCase{
			{"/endpoint1", expectation{
				Path:           "/anything",
				ResponseStatus: http.StatusOK,
			}},
			{"/endpoint2", expectation{
				Path:           "/anything",
				ResponseStatus: http.StatusOK,
			}},
			{"/endpoint3", expectation{
				Path:           "/unset/by/endpoint",
				ResponseStatus: http.StatusNotFound,
			}},
			{"/endpoint4", expectation{
				Path:           "/anything",
				ResponseStatus: http.StatusOK,
			}},
		} {
			t.Run(confFile+"_"+tc.reqPath, func(subT *testing.T) {
				helper := test.New(subT)
				shutdown, _ := newCouper(confPath, helper)
				defer shutdown()

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
					t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
				}

				if !reflect.DeepEqual(jsonResult, tc.exp) {
					t.Errorf("%q: %q:\nwant:\t%#v\ngot:\t%#v\npayload:\n%s", confFile, tc.reqPath, tc.exp, jsonResult, string(resBytes))
				}
			})
		}
	}
}

func TestHTTPServer_Endpoint_Evaluation_Inheritance_Backend_Block(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/endpoint_eval/08_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/"+
		strings.Replace(testBackend.Addr(), "http://", "", 1), nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusBadRequest {
		t.Error("Expected a bad request without required query param")
	}
}

func TestOpenAPIValidateConcurrentRequests(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/validation/01_couper.hcl", test.New(t))
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

func TestConfigBodyContent(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/config/01_couper.hcl", test.New(t))
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
				t.Errorf("%q: expected Status OK, got: %d", tc.path, res.StatusCode)
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
					t.Errorf("Expected Header %q value: %v, got: %v", k, v, res.Header[k])
				}
			}

			for k, v := range tc.query {
				if !reflect.DeepEqual(p.Query[k], v) {
					t.Errorf("Expected Query %q value: %v, got: %v", k, v, p.Query[k])
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
		{"/v5/not-exist", http.Header{}, http.StatusUnauthorized, "application/json", "access control error: ba1: credentials required"},
		{"/superadmin", http.Header{"Authorization": []string{"Basic OmFzZGY="}, "Auth": []string{"ba1", "ba4"}}, http.StatusOK, "application/json", ""},
		{"/superadmin", http.Header{}, http.StatusUnauthorized, "application/json", "access control error: ba1: credentials required"},
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

			message := getAccessControlMessages(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					t.Errorf("Expected error log: %q, actual: %#v", tc.wantErrLog, message)
				}
			} else {
				if message != tc.wantErrLog {
					t.Errorf("Expected error log message: %q, actual: %#v", tc.wantErrLog, message)
				}
			}

			if res.StatusCode != tc.status {
				t.Errorf("%q: expected Status %d, got: %d", tc.path, tc.status, res.StatusCode)
				return
			}

			if ct := res.Header.Get("Content-Type"); ct != tc.ct {
				t.Errorf("%q: expected content-type: %q, got: %q", tc.path, tc.ct, ct)
				return
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
					t.Errorf("Expected header %q, got nothing", k)
					break
				}
				if !reflect.DeepEqual(p.Headers[k], v) {
					t.Errorf("Expected header %q value: %v, got: %v", k, v, p.Headers[k])
				}
			}
		})
	}
}

func Test_LoadAccessControl(t *testing.T) {
	// Tests the config load with ACs and "error_handler" blocks...
	shutdown, _ := newCouper("testdata/integration/config/07_couper.hcl", test.New(t))
	defer shutdown()
}

func TestJWTAccessControl(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/03_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name       string
		path       string
		header     http.Header
		status     int
		expScope   string
		wantErrLog string
	}

	// RSA tokens created with server/testdata/integration/files/pkcs8.key
	rsaToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.AZ0gZVqPe9TjjjJO0GnlTvERBXhPyxW_gTn050rCoEkseFRlp4TYry7WTQ7J4HNrH3btfxaEQLtTv7KooVLXQyMDujQbKU6cyuYH6MZXaM0Co3Bhu0awoX-2GVk997-7kMZx2yvwIR5ypd1CERIbNs5QcQaI4sqx_8oGrjO5ZmOWRqSpi4Mb8gJEVVccxurPu65gPFq9esVWwTf4cMQ3GGzijatnGDbRWs_igVGf8IAfmiROSVd17fShQtfthOFd19TGUswVAleOftC7-DDeJgAK8Un5xOHGRjv3ypK_6ZLRonhswaGXxovE0kLq4ZSzumQY2hOFE6x_BbrR1WKtGw"

	for _, tc := range []testCase{
		{"no token", "/jwt", http.Header{}, http.StatusUnauthorized, "", "access control error: JWTToken: token required"},
		{"expired token", "/jwt", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjEyMzQ1Njc4OSwic2NvcGUiOlsiZm9vIiwiYmFyIl19.W2ziH_V33JkOA5ttQhzWN96RqxFydmx7GHY6G__U9HM"}}, http.StatusForbidden, "", "access control error: JWTToken: token is expired by "},
		{"valid token", "/jwt", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwic2NvcGUiOiJmb28gYmFyIiwiaWF0IjoxNTE2MjM5MDIyfQ.7wz7Z7IajfEpwYayfshag6tQVS0e0zZJyjAhuFC0L-E"}}, http.StatusOK, `["foo","bar"]`, ""},
		{"RSA JWT", "/jwt/rsa", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, http.StatusOK, "", ""},
		{"local RSA JWKS without kid", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEyMzQ1Njc4OTB9.V9skZUql-mHqwOzVdzamqAOWSx8fjEA-6py0nfxLRSl7h1bQvqUCWMZUAkMJK6RuJ3y5YAr8ZBXZsh4rwABp_3hitQitMXnV6nr5qfzVDE9-mdS4--Bj46-JlkHacNcK24qlnn_EXGJlzCj6VFgjObSy6geaTY9iDVF6EzjZkxc1H75XRlNYAMu-0KCGfKdte0qASeBKrWnoFNEpnXZ_jhqRRNVkaSBj7_HPXD6oPqKBQf6Jh6fGgdz6q4KNL-t-Qa2_eKc8tkrYNdTdxco-ufmmLiUQ_MzRAqowHb2LdsFJP9rN2QT8MGjRXqGvkCd0EsLfqAeCPkTXs1kN8LGlvw"}}, http.StatusForbidden, "", "access control error: JWKS: Missing \"kid\" in JOSE header"},
		{"local RSA JWKS with unsupported kid", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer eyJraWQiOiJyczI1Ni11bnN1cHBvcnRlZCIsImFsZyI6IlJTMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.wx1MkMgJhh6gnOvvrnnkRpEUDe-0KpKWw9ZIfDVHtGkuL46AktBgfbaW1ttB78wWrIW9OPfpLqKwkPizwfShoXKF9qN-6TlhPSWIUh0_kBHEj7H4u45YZXH1Ha-r9kGzly1PmLx7gzxUqRpqYnwo0TzZSEr_a8rpfWaC0ZJl3CKARormeF3tzW_ARHnGUqck4VjPfX50Ot6B5nool6qmsCQLLmDECIKBDzZicqdeWH7JPvRZx45R5ZHJRQpD3Z2iqVIF177Wj1C8q75Gxj2PXziIVKplmIUrKN-elYj3kBtJkDFneb384FPLuzsQZOR6HQmKXG2nA1WOfsblJSz3FA"}}, http.StatusForbidden, "", "access control error: JWKS: No matching RS256 JWK for kid \"rs256-unsupported\""},
		{"local RSA JWKS with non-parsable cert", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer eyJraWQiOiJyczI1Ni13cm9uZy1jZXJ0IiwiYWxnIjoiUlMyNTYiLCJ0eXAiOiJKV1QifQ.eyJzdWIiOjEyMzQ1Njc4OTB9.n--6mjzfnPKbaYAquBK3v6gsbmvEofSprk3jwWGSKPdDt2VpVOe8ZNtGhJj_3f1h86-wg-gEQT5GhJmsI47X9MJ70j74dqhXUF6w4782OljstP955whuSM9hJAIvUw_WV1sqtkiESA-CZiNJIBydL5YzV2nO3gfEYdy9EdMJ2ykGLRBajRxhShxsfaZykFKvvWpy1LbUc-gfRZ4q8Hs9B7b_9RGdbpRwBtwiqPPzhjC5O86vk7ZoiG9Gq7pg52yEkLqdN4a5QkfP8nNeTTMAsqPQL1-1TAC7rIGekoUtoINRR-cewPpZ_E7JVxXvBVvPe3gX_2NzGtXkLg5QDt6RzQ"}}, http.StatusForbidden, "", "access control error: JWKS: No matching RS256 JWK for kid \"rs256-wrong-cert\""},
		{"local RSA JWKS not found", "/jwks/rsa/not_found", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, http.StatusForbidden, "", "access control error: JWKS_not_found: Error loading JWKS: Status code 404"},
		{"local RSA JWKS", "/jwks/rsa", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, http.StatusOK, "", ""},
		{"local RSA JWKS with scope", "/jwks/rsa/scope", http.Header{"Authorization": []string{"Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTAsInNjb3BlIjpbImZvbyIsImJhciJdfQ.IFqIF_9ELXl3A-oy52G0Sg5f34ah3araOxFboskEw110nXdb_-UuxCnG0naFVFje7xvNrGbJgVAbBRX1v1I_to4BR8RzvIh2hi5IgBmqclIYsYbVWlEhsvjBhFR2b90Rz0APUdfgHp-nvgLB13jxm8f4TRr4ZDnvUQdZp3vI5PMj9optEmlZvexkNLDQLrBvoGCfVHodZyPQMLNVKp0TXWksPT-bw0E7Lq1GeYe2eU0GwHx8fugo2-v44dfCp0RXYYG6bI_Z-U3KZpvdj05n2_UDgTJFFm4c5i9UjILvlO73QJpMNi5eBjerm2alTisSCoiCtfgIgVsM8yHoomgarg"}}, http.StatusOK, `["foo","bar"]`, ""},
		{"remote RSA JWKS x5c", "/jwks/rsa/remote", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, http.StatusOK, "", ""},
		{"remote RSA JWKS x5c w/ backend", "/jwks/rsa/backend", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, http.StatusOK, "", ""},
		{"remote RSA JWKS x5c w/ backendref", "/jwks/rsa/backendref", http.Header{"Authorization": []string{"Bearer " + rsaToken}}, http.StatusOK, "", ""},
		{"remote RSA JWKS n, e", "/jwks/rsa/remote", http.Header{"Authorization": []string{"Bearer eyJraWQiOiJyczI1Ni1uZSIsImFsZyI6IlJTMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.aGOhlWQIZvnwoEZGDBYhkkEduIVa59G57x88L3fiLc1MuWbYS84nHEZnlPDuVJ3_BxdXr6-nZ8gpk1C9vfamDzkbvzbdcJ2FzmvAONm1II3_u5OTc6ZtpREDx9ohlIvkcOcalOUhQLqU5r2uik2bGSVV3vFDbqxQeuNzh49i3VgdtwoaryNYSzbg_Ki8dHiaFrWH-r2WCU08utqpFmNdr8oNw4Y5AYJdUW2aItxDbwJ6YLBJN0_6EApbXsNqiaNXkLws3cxMvczGKODyGGVCPENa-VmTQ41HxsXB-_rMmcnMw3_MjyIueWcjeP8BNvLYt1bKFWdU0NcYCkXvEqE4-g"}}, http.StatusOK, "", ""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			if val := tc.header.Get("Authorization"); val != "" {
				req.Header.Set("Authorization", val)
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				subT.Errorf("expected Status %d, got: %d", tc.status, res.StatusCode)
				return
			}

			message := getAccessControlMessages(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					subT.Errorf("Expected error log: %q, actual: %#v", tc.wantErrLog, message)
				}
			} else {
				if !strings.HasPrefix(message, tc.wantErrLog) {
					subT.Errorf("Expected error log message: %q, actual: %#v", tc.wantErrLog, message)
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

			if scopes := res.Header.Get("X-Scopes"); scopes != tc.expScope {
				subT.Errorf("expected scope: %q, actual: %q", tc.expScope, scopes)
				return
			}
		})
	}
}

func TestJWTAccessControlSourceConfig(t *testing.T) {
	helper := test.New(t)
	couperConfig, err := configload.LoadFile("testdata/integration/config/05_couper.hcl")
	helper.Must(err)

	log, _ := logrustest.NewNullLogger()
	ctx := context.TODO()

	expectedMsg := "configuration error: missing-source: token source is invalid"

	err = command.NewRun(ctx).Execute([]string{couperConfig.Filename}, couperConfig, log.WithContext(ctx))
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
		name string
		path string
	}

	for _, tc := range []testCase{
		{"separate jwt_signing_profile/jwt", "/separate"},
		{"self-signed jwt", "/self-signed"},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://back.end:8080%s/%s/create-jwt", tc.path, pid), nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				t.Errorf("%q: token request: unexpected status: %d", tc.name, res.StatusCode)
				return
			}

			token := res.Header.Get("X-Jwt")

			req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("http://back.end:8080%s/%s/jwt", tc.path, pid), nil)
			helper.Must(err)
			req.Header.Set("Authorization", "Bearer "+token)

			res, err = client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusOK {
				t.Errorf("%q: resource request: unexpected status: %d", tc.name, res.StatusCode)
				return
			}

			decoder := json.NewDecoder(res.Body)
			var claims map[string]interface{}
			err = decoder.Decode(&claims)
			helper.Must(err)

			if _, ok := claims["exp"]; !ok {
				t.Errorf("%q: missing exp claim: %#v", tc.name, claims)
				return
			}
			issclaim, ok := claims["iss"]
			if !ok {
				t.Errorf("%q: missing iss claim: %#v", tc.name, claims)
				return
			}
			if issclaim != "the_issuer" {
				t.Errorf("%q: unexpected iss claim: %q", tc.name, issclaim)
				return
			}
			pidclaim, ok := claims["pid"]
			if !ok {
				t.Errorf("%q: missing pid claim: %#v", tc.name, claims)
				return
			}
			if pidclaim != pid {
				t.Errorf("%q: unexpected pid claim: %q", tc.name, pidclaim)
				return
			}
		})
	}
}

func getAccessControlMessages(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Message != "" {
			return entry.Message
		}
	}

	return ""
}

func Test_Scope(t *testing.T) {
	h := test.New(t)
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/config/09_couper.hcl", test.New(t))
	defer shutdown()

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"scp": "a",
	})
	token, tokenErr := tok.SignedString([]byte("asdf"))
	h.Must(tokenErr)

	type testCase struct {
		name        string
		operation   string
		path        string
		authorize   bool
		status      int
		wantErrLog  string
		wantErrType string
	}

	for _, tc := range []testCase{
		{"unauthorized", http.MethodGet, "/foo", false, http.StatusUnauthorized, "access control error: myjwt: token required", "jwt_token_missing"},
		{"sufficient scope", http.MethodGet, "/foo", true, http.StatusNoContent, "", ""},
		{"additional scope required: insufficient scope", http.MethodPost, "/foo", true, http.StatusForbidden, `access control error: scope: required scope "foo" not granted`, "beta_insufficient_scope"},
		{"operation not permitted", http.MethodDelete, "/foo", true, http.StatusForbidden, "access control error: scope: operation DELETE not permitted", "beta_operation_denied"},
		{"additional scope required by *: insufficient scope", http.MethodGet, "/bar", true, http.StatusForbidden, `access control error: scope: required scope "more" not granted`, "beta_insufficient_scope"},
		{"no additional scope required: sufficient scope", http.MethodDelete, "/bar", true, http.StatusNoContent, "", ""},
	} {
		t.Run(fmt.Sprintf("%s_%s_%s", tc.name, tc.operation, tc.path), func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			req, err := http.NewRequest(tc.operation, "http://back.end:8080"+tc.path, nil)
			if tc.authorize {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				t.Errorf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
				return
			}

			message := getAccessControlMessages(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					t.Errorf("Expected error log: %q, actual: %#v", tc.wantErrLog, message)
				}
			} else {
				if !strings.HasPrefix(message, tc.wantErrLog) {
					t.Errorf("Expected error log message: %q, actual: %#v", tc.wantErrLog, message)
				}
				errorType := getAccessLogErrorType(hook)
				if errorType != tc.wantErrType {
					t.Errorf("Expected error type: %q, actual: %q", tc.wantErrType, errorType)
				}
			}
		})
	}
}

func getAccessLogUrl(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_access" && entry.Data["url"] != "" {
			if url, ok := entry.Data["url"].(string); ok {
				return url
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
		"Echo":              []string{"ECHO"},
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
		t.Run(tc.path[1:], func(st *testing.T) {
			helper := test.New(st)

			req, err := http.NewRequest(http.MethodGet, "http://protect.me:8080"+tc.path, nil)
			helper.Must(err)

			if tc.password != "" {
				req.SetBasicAuth("", tc.password)
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Errorf("Expected status: %d, got: %d", tc.expStatus, res.StatusCode)
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
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
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
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				t.Errorf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
				return
			}

			for k, v := range tc.header {
				if v1 := res.Header.Get(k); v1 != v {
					t.Errorf("%q: unexpected %s response header %#v, got: %#v", tc.name, k, v, v1)
					return
				}
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
					t.Errorf("expected a redirect response")
				}

				resp := logHook.LastEntry().Data["response"]
				fields := resp.(logging.Fields)
				headers := fields["headers"].(map[string]string)
				if headers["location"] != "https://couper.io/" {
					t.Errorf("expected location header log")
				}
			} else {
				helper.Must(err)
			}

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)
			helper.Must(res.Body.Close())

			if res.StatusCode != tc.expStatusCode {
				t.Errorf("%q: expected Status %d, got: %d", tc.path, tc.expStatusCode, res.StatusCode)
				return
			}

			if logHook.LastEntry().Data["status"] != tc.expStatusCode {
				t.Logf("%v", logHook.LastEntry())
				t.Errorf("Expected statusCode log: %d", tc.expStatusCode)
			}

			if len(resBytes) > 0 {
				b, exist := logHook.LastEntry().Data["response"].(logging.Fields)["bytes"]
				if !exist || b != len(resBytes) {
					t.Errorf("Want bytes log: %d\ngot:\t%v", len(resBytes), logHook.LastEntry())
				}
			}
		})
	}
}

func TestCORS_Configuration(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/config/06_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path             string
		origin           string
		expAllowedOrigin bool
	}

	for _, tc := range []testCase{
		{"/06_couper.hcl", "a.com", true},
		{"/spa/", "b.com", true},
		{"/api/", "c.com", true},
		{"/06_couper.hcl", "no.com", false},
		{"/spa/", "", false},
		{"/api/", "no.com", false},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodOptions, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			req.Header.Set("Access-Control-Request-Method", "GET")
			req.Header.Set("Access-Control-Request-Headers", "origin")
			req.Header.Set("Origin", tc.origin)

			res, err := client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != http.StatusNoContent {
				t.Errorf("%q: expected Status %d, got: %d", tc.path, http.StatusNoContent, res.StatusCode)
				return
			}

			if val, exist := res.Header["Access-Control-Allow-Origin"]; tc.expAllowedOrigin && (!exist || val[0] != tc.origin) {
				t.Errorf("Expected allowed origin resp, got: %v", val)
			}
		})
	}
}

func TestLog_Level(t *testing.T) {
	shutdown, hook := newCouper("testdata/integration/logging/01_couper.hcl", test.New(t))
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
		t.Errorf("expected Status %d, got: %d", 200, res.StatusCode)
		return
	}

	v1 := res.Header.Get("x-v-1")
	v2 := res.Header.Get("x-v-2")
	hv := res.Header.Get("x-hv")
	if v2 != v1 {
		t.Errorf("multiple calls to beta_oauth_verifier() must return the same value:\n\t%s\n\t%s", v1, v2)
	}
	s256 := oauth2.Base64urlSha256(v1)
	if hv != s256 {
		t.Errorf("call to internal_oauth_hashed_verifier() returns wrong value:\nactual:\t\t%s\nexpected:\t%s", hv, s256)
	}
	au, err := url.Parse(res.Header.Get("x-au-pkce"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("beta_oauth_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("beta_oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email" {
		t.Errorf("beta_oauth_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile email")
	}
	if auq.Get("code_challenge_method") != "S256" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "S256")
	}
	if auq.Get("code_challenge") != hv {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), hv)
	}
	if auq.Get("state") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("beta_oauth_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
	au, err = url.Parse(res.Header.Get("x-au-pkce-rel"))
	helper.Must(err)
	auq = au.Query()
	if auq.Get("redirect_uri") != "http://example.com:8080/oidc/callback" {
		t.Errorf("oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://example.com:8080/oidc/callback")
	}

	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/pkce", nil)
	helper.Must(err)

	res, err = client.Do(req)
	helper.Must(err)

	cv1_n := res.Header.Get("x-v-1")
	if cv1_n == v1 {
		t.Errorf("calls to beta_oauth_verifier() on different requests must not return the same value:\n\t%s\n\t%s", v1, cv1_n)
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
		t.Errorf("expected Status %d, got: %d", 200, res.StatusCode)
		return
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-state"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("beta_oauth_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("beta_oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile" {
		t.Errorf("beta_oauth_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile")
	}
	if auq.Get("code_challenge_method") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "")
	}
	if auq.Get("code_challenge") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), "")
	}
	if auq.Get("state") != hv {
		t.Errorf("beta_oauth_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), hv)
	}
	if auq.Get("nonce") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("beta_oauth_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
}

func TestOIDCPKCEFunctions(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/pkce", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Errorf("expected Status %d, got: %d", 200, res.StatusCode)
		return
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-pkce"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("beta_oauth_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("beta_oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email" {
		t.Errorf("beta_oauth_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile email")
	}
	if auq.Get("code_challenge_method") != "S256" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "S256")
	}
	if auq.Get("code_challenge") != hv {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), hv)
	}
	if auq.Get("state") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("beta_oauth_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
	au, err = url.Parse(res.Header.Get("x-au-pkce-rel"))
	helper.Must(err)
	auq = au.Query()
	if auq.Get("redirect_uri") != "http://example.com:8080/oidc/callback" {
		t.Errorf("oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://example.com:8080/oidc/callback")
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
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/csrf", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Errorf("expected Status %d, got: %d", 200, res.StatusCode)
		return
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-nonce"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("beta_oauth_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("beta_oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile" {
		t.Errorf("beta_oauth_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile")
	}
	if auq.Get("code_challenge_method") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "")
	}
	if auq.Get("code_challenge") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), "")
	}
	if auq.Get("state") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != hv {
		t.Errorf("beta_oauth_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), hv)
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("beta_oauth_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
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
			"userinfo_endpoint": "http://` + req.Host + `/userinfo",
			"code_challenge_methods_supported": ["S256"]
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/default", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Errorf("expected Status %d, got: %d", 200, res.StatusCode)
		return
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-default"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("beta_oauth_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("beta_oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email address" {
		t.Errorf("beta_oauth_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile email")
	}
	if auq.Get("code_challenge_method") != "S256" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "S256")
	}
	if auq.Get("code_challenge") != hv {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), hv)
	}
	if auq.Get("state") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), "")
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("beta_oauth_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
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
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/functions/03_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/default", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 200 {
		t.Errorf("expected Status %d, got: %d", 200, res.StatusCode)
		return
	}

	hv := res.Header.Get("x-hv")
	au, err := url.Parse(res.Header.Get("x-au-default"))
	helper.Must(err)
	auq := au.Query()
	if auq.Get("response_type") != "code" {
		t.Errorf("beta_oauth_authorization_url(): wrong response_type query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("response_type"), "code")
	}
	if auq.Get("redirect_uri") != "http://localhost:8085/oidc/callback" {
		t.Errorf("beta_oauth_authorization_url(): wrong redirect_uri query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("redirect_uri"), "http://localhost:8085/oidc/callback")
	}
	if auq.Get("scope") != "openid profile email address" {
		t.Errorf("beta_oauth_authorization_url(): wrong scope query param:\nactual:\t\t%s\nexpected:\t%s", auq.Get("scope"), "openid profile")
	}
	if auq.Get("code_challenge_method") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge_method:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge_method"), "")
	}
	if auq.Get("code_challenge") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong code_challenge:\nactual:\t\t%s\nexpected:\t%s", auq.Get("code_challenge"), "")
	}
	if auq.Get("state") != "" {
		t.Errorf("beta_oauth_authorization_url(): wrong state:\nactual:\t\t%s\nexpected:\t%s", auq.Get("state"), "")
	}
	if auq.Get("nonce") != hv {
		t.Errorf("beta_oauth_authorization_url(): wrong nonce:\nactual:\t\t%s\nexpected:\t%s", auq.Get("nonce"), hv)
	}
	if auq.Get("client_id") != "foo" {
		t.Errorf("beta_oauth_authorization_url(): wrong client_id:\nactual:\t\t%s\nexpected:\t%s", auq.Get("client_id"), "foo")
	}
}
