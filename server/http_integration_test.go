package server_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/avenga/couper/config/configload"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/internal/test"
)

var (
	testBackend    *test.Backend
	testWorkingDir string
	testProxyAddr  = "http://127.0.0.1:9999"
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

	log, hook := logrustest.NewNullLogger()

	ctx, cancel := context.WithCancel(context.Background())
	cancelFn := func() {
		cancel()
		time.Sleep(time.Second / 2)
	}
	shutdownFn := func() {
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

	//log.Out = os.Stdout

	go func() {
		if err := command.NewRun(ctx).Execute([]string{file}, couperConfig, log.WithContext(ctx)); err != nil {
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
				expectation{http.StatusInternalServerError, []byte("<html>1002</html>"), http.Header{"Couper-Error": {`1002 - "Configuration failed"`}}, ""},
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
				expectation{http.StatusInternalServerError, []byte("<html>1002</html>"), nil, ""},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/b"},
				expectation{http.StatusOK, []byte(`<html lang="en">index B</html>`), nil, "file"},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/"},
				expectation{http.StatusInternalServerError, []byte("<html>1002</html>"), nil, ""},
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
				expectation{http.StatusInternalServerError, []byte("<html>1002</html>"), http.Header{"Couper-Error": {`1002 - "Configuration failed"`}}, ""},
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
				expectation{http.StatusNotFound, []byte(`{"code": 4001}`), http.Header{"Content-Type": {"application/json"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1/connect-error/"}, // in this case proxyconnect fails
				expectation{http.StatusBadGateway, []byte(`{"code": 7001}`), http.Header{"Content-Type": {"application/json"}}, "api"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/v1x"},
				expectation{http.StatusInternalServerError, []byte(`<html>1002</html>`), http.Header{"Content-Type": {"text/html"}}, ""},
			},
		}},
		{"api/02_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusInternalServerError, []byte("<html>1002</html>"), http.Header{"Couper-Error": {`1002 - "Configuration failed"`}}, ""},
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
				expectation{http.StatusNotFound, []byte(`{"code": 4001}`), http.Header{"Content-Type": {"application/json"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://couper.io:9898/v2/not-found"},
				expectation{http.StatusNotFound, []byte(`{"code": 4001}`), http.Header{"Content-Type": {"application/json"}}, ""},
			},
			{
				testRequest{http.MethodGet, "http://example.com:9898/v3/not-found"},
				expectation{http.StatusNotFound, []byte(`{"code": 4001}`), http.Header{"Content-Type": {"application/json"}}, ""},
			},
		}},
		{"vhosts/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/notfound"},
				expectation{http.StatusNotFound, []byte("<html>3001</html>"), http.Header{"Couper-Error": {`3001 - "Files route not found"`}}, ""},
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
				expectation{http.StatusNotFound, []byte("<html>3001</html>"), http.Header{"Couper-Error": {`3001 - "Files route not found"`}}, ""},
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

				resBytes, err := ioutil.ReadAll(res.Body)
				helper.Must(err)

				_ = res.Body.Close()

				if res.StatusCode != rc.exp.status {
					t.Errorf("Expected statusCode %d, got %d", rc.exp.status, res.StatusCode)
				}

				for k, v := range rc.exp.header {
					if !reflect.DeepEqual(res.Header[k], v) {
						t.Errorf("Exptected headers:\nWant:\t%#v\nGot:\t%#v\n", v, res.Header[k])
					}
				}

				if rc.exp.body != nil && bytes.Compare(resBytes, rc.exp.body) != 0 {
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

	resBytes, err := ioutil.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	if `<html lang="en">index B</html>` != string(resBytes) {
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

	resBytes, err := ioutil.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	if `<html>1002</html>` != string(resBytes) {
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

	os.Setenv("COUPER_XFH", "true")
	confPath := path.Join("testdata/integration", "files/02_couper.hcl")
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()
	os.Setenv("COUPER_XFH", "")

	helper := test.New(t)
	logHook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:9898/b", nil)
	helper.Must(err)

	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Host", "example.com.")
	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := ioutil.ReadAll(res.Body)
	helper.Must(err)

	_ = res.Body.Close()

	if `<html lang="en">index B</html>` != string(resBytes) {
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

			resBytes, err := ioutil.ReadAll(body)
			helper.Must(err)

			srcBytes, err := ioutil.ReadFile(filepath.Join(testWorkingDir, "testdata/integration/files/htdocs_c_gzip"+tc.path))
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
		tc := tc
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

			resBytes, err := ioutil.ReadAll(res.Body)
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

			resBytes, err := ioutil.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			jsonResult.Headers.Del("User-Agent")
			jsonResult.Headers.Del("X-Forwarded-For")

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestHTTPServer_QueryEncoding(t *testing.T) {
	client := newClient()

	config := "testdata/integration/endpoint_eval/10_couper.hcl"

	type expectation struct {
		RawQuery string
	}

	helper := test.New(t)
	shutdown, _ := newCouper(config, helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080?a=a%20a&x=x+x", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := ioutil.ReadAll(res.Body)
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

	config := "testdata/integration/config/02_couper.hcl"

	helper := test.New(t)
	shutdown, _ := newCouper(config, helper)
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

func TestHTTPServer_TrailingSlash(t *testing.T) {
	client := newClient()

	config := "testdata/integration/endpoint_eval/11_couper.hcl"

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
			shutdown, _ := newCouper(config, helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := ioutil.ReadAll(res.Body)
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

			resBytes, err := ioutil.ReadAll(res.Body)
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

				resBytes, err := ioutil.ReadAll(res.Body)
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

			b, err := ioutil.ReadAll(res.Body)

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

	shutdown, _ := newCouper("testdata/integration/config/03_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path   string
		header http.Header
		status int
	}

	for _, tc := range []testCase{
		{"/v1", http.Header{"Auth": []string{"ba1"}}, http.StatusOK},
		// TODO: Can a disabled auth being enabled again?
		//{"/v1", http.Header{"Authorization": []string{"Basic OmFzZGY="}, "Auth": []string{"ba1"}}, http.StatusOK},
		//{"/v1", http.Header{"Auth": []string{}}, http.StatusUnauthorized},
		{"/v2", http.Header{"Authorization": []string{"Basic OmFzZGY="}, "Auth": []string{"ba1", "ba2"}}, http.StatusOK}, // minimum ':'
		{"/v2", http.Header{}, http.StatusUnauthorized},
		{"/v3", http.Header{}, http.StatusOK},
		{"/status", http.Header{}, http.StatusOK},
		{"/superadmin", http.Header{"Authorization": []string{"Basic OmFzZGY="}, "Auth": []string{"ba1", "ba4"}}, http.StatusOK},
		{"/superadmin", http.Header{}, http.StatusUnauthorized},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			helper := test.New(subT)
			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			if val := tc.header.Get("Authorization"); val != "" {
				req.Header.Set("Authorization", val)
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				t.Errorf("%q: expected Status %d, got: %d", tc.path, tc.status, res.StatusCode)
				return
			}

			if ct := res.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("%q: unexpected content-type: %q", tc.path, ct)
				return
			}

			b, err := ioutil.ReadAll(res.Body)

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

func TestWrapperHiJack_WebsocketUpgrade(t *testing.T) {
	t.Skip("TODO fix hijack and endpoint handling for ws")
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

	p := make([]byte, 77)
	_, err = conn.Read(p)
	helper.Must(err)

	if !bytes.Equal(p, []byte("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")) {
		t.Errorf("Expected 101 status and related headers, got:\n%q", string(p))
	}

	n, err := conn.Write([]byte("ping"))
	helper.Must(err)

	if n != 4 {
		t.Errorf("Expected 4 written bytes for 'ping', got: %d", n)
	}

	p = make([]byte, 4)
	_, err = conn.Read(p)
	helper.Must(err)

	if !bytes.Equal(p, []byte("pong")) {
		t.Errorf("Expected pong answer, got: %q", string(p))
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
		{"/xxx", expectation{
			Path: "/xxx",
		}},
		{"/yyy", expectation{
			Path: "/yyy",
		}},
		{"/zzz", expectation{
			Path: "/zzz",
		}},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := ioutil.ReadAll(res.Body)
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
