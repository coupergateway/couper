package server_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"text/template"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

func TestHTTPServer_ServeHTTP_Files(t *testing.T) {
	expectedAPIHost := "test.couper.io"
	originBackend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Host != expectedAPIHost {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer originBackend.Close()

	tpl, err := template.ParseFiles("testdata/file_serving/conf_test.hcl")
	if err != nil {
		t.Fatal(err)
	}
	confBytes := &bytes.Buffer{}
	err = tpl.Execute(confBytes, map[string]string{
		"origin":   "http://" + originBackend.Listener.Addr().String(),
		"hostname": expectedAPIHost,
	})
	if err != nil {
		t.Fatal(err)
	}

	log, _ := logrustest.NewNullLogger()
	// log.Out = os.Stdout

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.LoadBytes(confBytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	errorPageContent, err := ioutil.ReadFile(conf.Server[0].Files.ErrorFile)
	if err != nil {
		t.Fatal(err)
	}

	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	if err != nil {
		t.Fatal(err)
	}

	conf.Addr = ":"
	gw := server.New(ctx, log.WithContext(ctx), conf)
	gw.Listen()
	defer gw.Close()

	connectClient := http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp4", gw.Addr())
		},
	}}

	for _, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", []byte("<title>500 Configuration failed</title>"), http.StatusInternalServerError},
		{"/apps/", []byte("<title>500 Configuration failed</title>"), http.StatusInternalServerError},
		{"/apps/shiny-product/", errorPageContent, http.StatusNotFound},
		{"/apps/shiny-product/assets/", errorPageContent, http.StatusNotFound},
		{"/apps/shiny-product/app/", spaContent, http.StatusOK},
		{"/apps/shiny-product/app/sub", spaContent, http.StatusOK},
		{"/apps/shiny-product/api/", nil, http.StatusNoContent},
		{"/apps/shiny-product/api/foo%20bar:%22baz%22", []byte(`"/apps/shiny-product/api/foo%20bar:%22baz%22"`), 404},
	} {
		res, err := connectClient.Get("http://example.com" + testCase.path)
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != testCase.expectedStatus {
			t.Errorf("Expected status %d, got %d", testCase.expectedStatus, res.StatusCode)
		}

		result := &bytes.Buffer{}
		_, err = io.Copy(result, res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(result.Bytes(), testCase.expectedBody) {
			t.Errorf("Expected body:\n%s\ngot:\n%s", string(testCase.expectedBody), result.String())
		}
	}
}

func TestHTTPServer_ServeHTTP_Files2(t *testing.T) {
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

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("testdata/file_serving")

	tpl, err := template.ParseFiles("conf_fileserving.hcl")
	if err != nil {
		t.Fatal(err)
	}
	confBytes := &bytes.Buffer{}
	err = tpl.Execute(confBytes, map[string]string{
		"origin":   "http://" + originBackend.Listener.Addr().String(),
		"hostname": expectedAPIHost,
	})
	if err != nil {
		t.Fatal(err)
	}

	log, _ := logrustest.NewNullLogger()
	// log.Out = os.Stdout

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.LoadBytes(confBytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	if err != nil {
		t.Fatal(err)
	}

	conf.Addr = ":"
	couper := server.New(ctx, log.WithContext(ctx), conf)
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

	for _, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", []byte("<title>404 FilesRouteNotFound</title>"), 404},
		{"/dir", nil, 302},
		{"/dir/", []byte("<html>this is dir/index.html</html>\n"), 200},
		{"/robots.txt", []byte("Disallow: /secret\n"), 200},
		{"/foo bar.txt", []byte("foo-and-bar\n"), 200},
		{"/foo%20bar.txt", []byte("foo-and-bar\n"), 200},
		{"/favicon.ico", []byte("<title>404 FilesRouteNotFound</title>"), 404},
		{"/app", spaContent, 200},
		{"/app/", spaContent, 200},
		{"/app/bla", spaContent, 200},
		{"/app/bla/foo", spaContent, 200},
		{"/api/foo/bar", []byte("/bar"), 200},
		//FIXME:
		//{"/api", content500.Bytes(), 500},
	} {
		res, err := connectClient.Get("http://example.com" + testCase.path)
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != testCase.expectedStatus {
			t.Errorf("Expected status %d, got %d", testCase.expectedStatus, res.StatusCode)
		}

		result := &bytes.Buffer{}
		_, err = io.Copy(result, res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(result.Bytes(), testCase.expectedBody) {
			t.Errorf("Expected body:\n%s\ngot:\n%s", string(testCase.expectedBody), result.String())
		}
	}
}
