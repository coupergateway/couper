package server_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/internal/test"
)

var (
	testBackend    *test.Backend
	testWorkingDir string
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

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testWorkingDir = wd
}

func teardown() {
	println("INTEGRATION: close test backend...")
	testBackend.Close()
}

func newCouper(file string, helper *test.Helper) (func(), *logrustest.Hook) {
	_, err := runtime.SetWorkingDirectory(filepath.Join(testWorkingDir, file))
	helper.Must(err)

	gatewayConf, err := config.LoadFile(path.Base(file))
	helper.Must(err)

	// replace all origins with our test backend addr
	// TODO: limitation: no support for inline origin changes
	if gatewayConf.Definitions != nil {
		for _, backend := range gatewayConf.Definitions.Backend {
			backendSchema := config.Backend{}.Schema(false)
			inlineSchema := config.Backend{}.Schema(true)
			backendSchema.Attributes = append(backendSchema.Attributes, inlineSchema.Attributes...)
			content, diags := backend.Options.Content(backendSchema)
			if diags != nil && diags.HasErrors() {
				helper.Must(diags)
			}

			if _, ok := content.Attributes["origin"]; !ok {
				helper.Must(fmt.Errorf("backend requires an origin value"))
			}

			content.Attributes["origin"].Expr = hcltest.MockExprLiteral(cty.StringVal(testBackend.Addr()))
			backend.Options = hcltest.MockBody(content)
		}
	}

	log, hook := logrustest.NewNullLogger()

	ctx, cancel := context.WithCancel(test.NewContext(context.Background()))
	cancelFn := func() {
		cancel()
		time.Sleep(time.Second / 2)
	}
	//log.Out = os.Stdout
	go func() {
		helper.Must(command.NewRun(ctx).Execute([]string{file}, gatewayConf, log.WithContext(ctx)))
	}()
	time.Sleep(time.Second / 2)

	for _, entry := range hook.AllEntries() {
		if entry.Level < logrus.InfoLevel {
			helper.Must(fmt.Errorf("error: %#v: %s", entry.Data, entry.Message))
		}
	}

	hook.Reset() // no startup logs
	return cancelFn, hook
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
		},
	}
}

func cleanup(shutdown func(), t *testing.T) {
	shutdown()

	err := os.Chdir(testWorkingDir)
	if err != nil {
		t.Fatal(err)
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
				testRequest{http.MethodGet, "http://anyserver:8080/v1/connect-error/"},
				expectation{http.StatusBadGateway, []byte(`{"code": 4002}`), http.Header{"Content-Type": {"application/json"}}, "api"},
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
		shutdown, logHook := newCouper(confPath, test.New(t))

		for _, rc := range testcase.requests {
			t.Run(testcase.fileName+" "+rc.req.method+"|"+rc.req.url, func(subT *testing.T) {
				helper := test.New(subT)
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

		cleanup(shutdown, t)
	}
}

func TestHTTPServer_HostHeader(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration", "files/02_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))

	t.Run("Test", func(subT *testing.T) {
		helper := test.New(subT)

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
	})

	cleanup(shutdown, t)
}

func TestHTTPServer_HostHeader2(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/integration", "api/03_couper.hcl")
	shutdown, logHook := newCouper(confPath, test.New(t))

	t.Run("Test", func(subT *testing.T) {
		helper := test.New(subT)

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
	})

	cleanup(shutdown, t)
}

func TestHTTPServer_XFHHeader(t *testing.T) {
	client := newClient()

	os.Setenv("COUPER_XFH", "true")
	confPath := path.Join("testdata/integration", "files/02_couper.hcl")
	shutdown, logHook := newCouper(confPath, test.New(t))
	os.Setenv("COUPER_XFH", "")

	t.Run("Test", func(subT *testing.T) {
		helper := test.New(subT)
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
		}
	})

	cleanup(shutdown, t)
}
