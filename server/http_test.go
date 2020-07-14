package server_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

func TestHTTPServer_ServeHTTP_Files(t *testing.T) {
	conf := &config.Gateway{Addr: ":", Server: []*config.Server{
		{
			BasePath: "/apps/shiny-product",
			Domains: []string{"example.com"},
			Name:    "Test_Files",
			Files: &config.Files{
				DocumentRoot: "testdata/file_serving/htdocs", ErrorFile: "testdata/file_serving/error.html",
			},
			Api: &config.Api{
				BasePath: "/api",
			},
			Spa: &config.Spa{
				BasePath: "/app",
				BootstrapFile: "testdata/file_serving/htdocs/spa.html",
				Paths:         []string{"/app/**"},
			},
		},
	}}

	errorPageContent, err := ioutil.ReadFile(conf.Server[0].Files.ErrorFile)
	if err != nil {
		t.Fatal(err)
	}

	spaContent, err := ioutil.ReadFile(conf.Server[0].Spa.BootstrapFile)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, _ := logrustest.NewNullLogger()
	log.Out = os.Stdout

	gw := server.New(ctx, log.WithContext(ctx), config.Load(conf, log.WithContext(ctx)))
	gw.Listen()

	for _, testCase := range []struct{
		path string
		expectedBody []byte
		expectedStatus int
	}{
		{"/", errorPageContent, http.StatusNotFound},
		{"/apps/", errorPageContent, http.StatusNotFound},
		{"/apps/shinyproduct/", errorPageContent, http.StatusNotFound},
		{"/apps/shinyproduct/assets/", errorPageContent, http.StatusOK},
		{"/apps/shinyproduct/app/", spaContent, http.StatusOK},
		{"/apps/shinyproduct/app/sub", spaContent, http.StatusOK},
		{"/apps/shinyproduct/api/", errorPageContent, http.StatusOK},
	}{
		res, err := http.DefaultClient.Get("http://" + gw.Addr() + testCase.path)
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
