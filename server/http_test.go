package server_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"text/template"
	"time"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server"
)

func TestHTTPServer_ServeHTTP_Files(t *testing.T) {
	helper := test.New(t)

	currentDir, err := os.Getwd()
	helper.Must(err)
	defer helper.Must(os.Chdir(currentDir))

	expectedAPIHost := "test.couper.io"
	originBackend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Host != expectedAPIHost {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer originBackend.Close()

	helper.Must(os.Chdir("testdata/file_serving"))

	tpl, err := template.ParseFiles("conf_test.hcl")
	helper.Must(err)

	confBytes := &bytes.Buffer{}
	err = tpl.Execute(confBytes, map[string]string{
		"origin":   "http://" + originBackend.Listener.Addr().String(),
		"hostname": expectedAPIHost,
	})
	helper.Must(err)

	log, _ := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpConf := runtime.NewHTTPConfig(nil)
	httpConf.ListenPort = 0 // random

	conf, err := config.LoadBytes(confBytes.Bytes())
	helper.Must(err)

	srvConf, err := runtime.NewServerConfiguration(conf, httpConf, log.WithContext(nil))
	helper.Must(err)

	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	helper.Must(err)

	port := runtime.Port(httpConf.ListenPort)
	gw := server.New(ctx, log.WithContext(ctx), httpConf, port, srvConf.PortOptions[port])
	gw.Listen()
	defer gw.Close()

	connectClient := http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp4", gw.Addr())
		},
	}}

	for i, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", []byte("<html><body><h1>1002: Configuration failed: My custom error template</h1></body></html>"), http.StatusInternalServerError},
		{"/apps/", []byte("<html><body><h1>1002: Configuration failed: My custom error template</h1></body></html>"), http.StatusInternalServerError},
		{"/apps/shiny-product/", []byte("<html><body><h1>3001: Files route not found: My custom error template</h1></body></html>"), http.StatusNotFound},
		{"/apps/shiny-product/assets/", []byte("<html><body><h1>3001: Files route not found: My custom error template</h1></body></html>"), http.StatusNotFound},
		{"/apps/shiny-product/app/", spaContent, http.StatusOK},
		{"/apps/shiny-product/app/sub", spaContent, http.StatusOK},
		{"/apps/shiny-product/api/", nil, http.StatusNoContent},
		{"/apps/shiny-product/api/foo%20bar:%22baz%22", []byte(`{"code": 4001}`), 404},
	} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com:%s%s", port, testCase.path), nil)
		helper.Must(err)

		req.Header.Set("Accept-Encoding", "br")
		res, err := connectClient.Do(req)
		helper.Must(err)

		if res.StatusCode != testCase.expectedStatus {
			t.Errorf("%.2d: expected status %d, got %d", i+1, testCase.expectedStatus, res.StatusCode)
		}

		result, err := ioutil.ReadAll(res.Body)
		helper.Must(err)
		helper.Must(res.Body.Close())

		if !bytes.Contains(result, testCase.expectedBody) {
			t.Errorf("%.2d: expected body should contain:\n%s\ngot:\n%s", i+1, string(testCase.expectedBody), string(result))
		}
	}

	helper.Must(os.Chdir(currentDir)) // defer for error cases, would be to late for normal exit
}

func TestHTTPServer_ServeHTTP_Files2(t *testing.T) {
	helper := test.New(t)

	currentDir, err := os.Getwd()
	helper.Must(err)
	defer helper.Must(os.Chdir(currentDir))

	expectedAPIHost := "test.couper.io"
	originBackend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Host != expectedAPIHost {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(req.URL.Path))
	}))
	defer originBackend.Close()

	helper.Must(os.Chdir("testdata/file_serving"))

	tpl, err := template.ParseFiles("conf_fileserving.hcl")
	helper.Must(err)

	confBytes := &bytes.Buffer{}
	err = tpl.Execute(confBytes, map[string]string{
		"origin":   "http://" + originBackend.Listener.Addr().String(),
		"hostname": expectedAPIHost,
	})
	helper.Must(err)

	log, _ := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpConf := runtime.NewHTTPConfig(nil)
	httpConf.ListenPort = 0 // random

	conf, err := config.LoadBytes(confBytes.Bytes())
	helper.Must(err)

	error404Content := []byte("<html><body><h1>3001: Files route not found: My custom error template</h1></body></html>")
	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	helper.Must(err)

	srvConf, err := runtime.NewServerConfiguration(conf, httpConf, log.WithContext(nil))
	helper.Must(err)
	port := runtime.Port(httpConf.ListenPort)

	couper := server.New(ctx, log.WithContext(ctx), httpConf, port, srvConf.PortOptions[port])
	couper.Listen()
	defer couper.Close()

	connectClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp4", couper.Addr())
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for i, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		// spa path /
		{"/", spaContent, 200},
		// 404 check that spa /dir/** rule doesn't match here
		{"/dirdoesnotexist", error404Content, 404},
		{"/dir:", error404Content, 404},
		{"/dir.txt", error404Content, 404},
		// dir w/ index in files
		{"/dir", nil, 302},
		// dir/ w/ index in files
		{"/dir/", []byte("<html>this is dir/index.html</html>\n"), 200},
		// dir w/o index in files
		{"/assets/noindex", error404Content, 404},
		{"/assets/noindex/", error404Content, 404},
		{"/assets/noindex/file.txt", []byte("foo\n"), 200},
		// dir w/o index in spa
		{"/dir/noindex", spaContent, 200},
		// file > spa
		{"/dir/noindex/otherfile.txt", []byte("bar\n"), 200},
		{"/robots.txt", []byte("Disallow: /secret\n"), 200},
		{"/foo bar.txt", []byte("foo-and-bar\n"), 200},
		{"/foo%20bar.txt", []byte("foo-and-bar\n"), 200},
		{"/favicon.ico", error404Content, 404},
		{"/app", spaContent, 200},
		{"/app/", spaContent, 200},
		{"/app/bla", spaContent, 200},
		{"/app/bla/foo", spaContent, 200},
		{"/api/foo/bar", []byte("/bar"), 200},
		//FIXME:
		//{"/api", content500.Bytes(), 500},
	} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com:%s%s", port, testCase.path), nil)
		helper.Must(err)

		req.Header.Set("Accept-Encoding", "br")
		res, err := connectClient.Do(req)
		helper.Must(err)

		if res.StatusCode != testCase.expectedStatus {
			t.Fatalf("%.2d: expected status for path %q %d, got %d", i+1, testCase.path, testCase.expectedStatus, res.StatusCode)
		}

		result, err := ioutil.ReadAll(res.Body)
		helper.Must(err)
		helper.Must(res.Body.Close())

		if !bytes.Contains(result, testCase.expectedBody) {
			t.Errorf("%.2d: expected body for path %q:\n%s\ngot:\n%s", i+1, testCase.path, string(testCase.expectedBody), string(result))
		}
	}
	helper.Must(os.Chdir(currentDir)) // defer for error cases, would be to late for normal exit
}

func TestHTTPServer_ServeHTTP_UUID_Option(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		formatOption string
		expRegex     *regexp.Regexp
	}

	for _, testcase := range []testCase{
		{"common", regexp.MustCompile(`^[0123456789abcdefghijklmnopqrstuv]{20}$`)},
		{"uuid4", regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)},
	} {
		t.Run(testcase.formatOption, func(subT *testing.T) {
			log, hook := logrustest.NewNullLogger()
			conf := *runtime.DefaultHTTP
			conf.RequestIDFormat = testcase.formatOption
			srv := server.New(context.Background(), log, &conf, 0, nil)
			srv.Listen()
			defer srv.Close()

			req := httptest.NewRequest(http.MethodGet, "http://"+srv.Addr()+"/", nil)
			req.RequestURI = ""
			req.Header.Set("Accept-Encoding", "br")

			hook.Reset()

			_, err := http.DefaultClient.Do(req)
			helper.Must(err)
			time.Sleep(time.Millisecond * 10) // log hook needs some time?

			if entry := hook.LastEntry(); entry == nil {
				t.Error("Expected a log entry")
			} else if !testcase.expRegex.MatchString(entry.Data["uid"].(string)) {
				t.Errorf("Expected a %q uid format, got %#v", testcase.formatOption, entry.Data["uid"])
			}
		})
	}
}
