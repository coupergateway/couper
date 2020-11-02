package server_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/test"
)

var (
	testBackend    *test.Backend
	testWorkingDir string

	defaultErrorTpl     = []byte("<html>{{.error_code}}</html>")
	defaultJSONErrorTpl = []byte(`{"code": {{.error_code}}}`)
	tmpErrTpl           errors.Template
	tmpJSONErrTpl       errors.Template
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	println("create test backend...")
	testBackend = test.NewBackend()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testWorkingDir = wd
	println("working directory: ", testWorkingDir)

	tmpErrTpl = *errors.DefaultHTML
	tmpJSONErrTpl = *errors.DefaultHTML

	testTpl, _ := errors.NewTemplate("text/html", defaultErrorTpl)
	errors.DefaultHTML = testTpl

	testApiTpl, _ := errors.NewTemplate("text/html", defaultJSONErrorTpl)
	errors.DefaultJSON = testApiTpl
}

func teardown() {
	println("close test backend...")
	testBackend.Close()

	errors.DefaultHTML = &tmpErrTpl
	errors.DefaultJSON = &tmpJSONErrTpl
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
			backend.Origin = testBackend.Addr()
		}
	}

	log, hook := logrustest.NewNullLogger()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		helper.Must(command.NewRun(ctx).Execute([]string{file}, gatewayConf, log.WithContext(context.Background())))
	}()
	time.Sleep(time.Second / 2)
	hook.Reset() // no startup logs
	return cancel, hook
}

func TestHTTPServer_ServeHTTP(t *testing.T) {
	helper := test.New(t)

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

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				_, port, _ := net.SplitHostPort(addr)
				if port != "" {
					return net.Dial("tcp4", "127.0.0.1:"+port)
				}
				return net.Dial("tcp4", "127.0.0.1")
			},
		},
	}

	for _, testcase := range []testCase{
		{"spa/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusOK, []byte(`<html><body><title>1.0</title></body></html>`), nil, "spa"},
			},
			{
				testRequest{http.MethodGet, "http://anyserver:8080/app"},
				expectation{http.StatusInternalServerError, []byte("<html>1002</html>"), http.Header{"Couper-Error": {`1002 - "Configuration failed"`}}, "spa"},
			},
		}},
		{"files/01_couper.hcl", []requestCase{
			{
				testRequest{http.MethodGet, "http://anyserver:8080/"},
				expectation{http.StatusOK, []byte(`<html><body><title>1.0</title></body></html>`), nil, "files"},
			},
		}},
	} {
		cancelFn, logHook := newCouper(path.Join("testdata/integration", testcase.fileName), helper)

		for _, rc := range testcase.requests {

			logHook.Reset()

			t.Run(testcase.fileName+" "+rc.req.method+"|"+rc.req.url, func(subT *testing.T) {
				req, err := http.NewRequest(rc.req.method, rc.req.url, nil)
				if err != nil {
					cancelFn()
					subT.Fatal(err)
				}

				res, err := client.Do(req)
				if err != nil {
					cancelFn()
					subT.Fatal(err)
				}

				resBytes, err := ioutil.ReadAll(res.Body)
				if err != nil {
					cancelFn()
					subT.Fatal(err)
				}

				_ = res.Body.Close()

				if res.StatusCode != rc.exp.status {
					t.Errorf("Expected statusCode %d, got %d", rc.exp.status, res.StatusCode)
				}

				for k, v := range rc.exp.header {
					if !reflect.DeepEqual(res.Header[k], v) {
						t.Errorf("Exptected headers:\nWant:\t%#v\nGot:\t%#v\n", v, res.Header[k])
					}
				}

				if bytes.Compare(resBytes, rc.exp.body) != 0 {
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

		cancelFn()
	}
}
