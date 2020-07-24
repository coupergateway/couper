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

	"go.avenga.cloud/couper/gateway/assets"
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

	conf := config.LoadBytes(confBytes.Bytes(), log.WithContext(ctx))

	errorPageContent, err := ioutil.ReadFile(conf.Server[0].Files.ErrorFile)
	if err != nil {
		t.Fatal(err)
	}

	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	if err != nil {
		t.Fatal(err)
	}

	gw := server.New(ctx, log.WithContext(ctx), conf)
	gw.Listen()

	connectClient := http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp4", gw.Addr())
		},
	}}

	content500, err := assets.Assets.Open("500.html")

	for _, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", content500.Bytes(), http.StatusInternalServerError},
		{"/apps/", content500.Bytes(), http.StatusInternalServerError},
		{"/apps/shiny-product/", errorPageContent, http.StatusNotFound},
		{"/apps/shiny-product/assets/", errorPageContent, http.StatusNotFound},
		{"/apps/shiny-product/app/", spaContent, http.StatusOK},
		{"/apps/shiny-product/app/sub", spaContent, http.StatusOK},
		{"/apps/shiny-product/api/", nil, http.StatusNoContent},
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
		if !bytes.Equal(testCase.expectedBody, result.Bytes()) {
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

	conf := config.LoadBytes(confBytes.Bytes(), log.WithContext(ctx))

	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	if err != nil {
		t.Fatal(err)
	}

	couper := server.New(ctx, log.WithContext(ctx), conf)
	couper.Listen()
	//defer couper.Close()

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

	content404, err := assets.Assets.Open("404.html")
	// content500, err := assets.Assets.Open("500.html")

	for _, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", content404.Bytes(), 404},
		{"/dir", nil, 302},
		{"/dir/", []byte("<html>this is dir/index.html</html>\n"), 200},
		{"/robots.txt", []byte("Disallow: /secret\n"), 200},
		{"/favicon.ico", content404.Bytes(), 404},
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
		if !bytes.Equal(testCase.expectedBody, result.Bytes()) {
			t.Errorf("Expected body:\n%s\ngot:\n%s", string(testCase.expectedBody), result.String())
		}
	}
}
