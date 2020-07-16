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
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"go.avenga.cloud/couper/gateway/backend"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

func TestHTTPServer_ServeHTTP_Files(t *testing.T) {
	originBackend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		println(req.Host)
		rw.WriteHeader(http.StatusNoContent)
	}))
	backendRemain := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(backend.Proxy{OriginAddress: originBackend.Listener.Addr().String(), OriginHost: "muh.de"}, backendRemain.Body())

	conf := &config.Gateway{Addr: ":", Server: []*config.Server{
		{
			BasePath: "/apps/shiny-product",
			Domains:  []string{"example.com"},
			Name:     "Test_Files",

			Files: &config.Files{
				DocumentRoot: "testdata/file_serving/htdocs", ErrorFile: "testdata/file_serving/error.html",
			},
			Api: &config.Api{
				Backend: []*config.Backend{
					{Options: hcl.MergeFiles([]*hcl.File{{Bytes: backendRemain.Bytes()}}), Kind: "proxy", Name: "example"},
				},
				BasePath: "/api",
				Endpoint: []*config.Endpoint{
					{Pattern: "/",
						Options: hcl.EmptyBody(),
						Backend: "example",
					},
				},
			},
			Spa: &config.Spa{
				BasePath:      "/app",
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

	connectClient := http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp4", gw.Addr())
		},
		ResponseHeaderTimeout: time.Second,
	}}

	for _, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", errorPageContent, http.StatusNotFound},
		{"/apps/", errorPageContent, http.StatusNotFound},
		{"/apps/shiny-product/", errorPageContent, http.StatusNotFound},
		{"/apps/shiny-product/assets/", errorPageContent, http.StatusOK},
		{"/apps/shiny-product/app/", spaContent, http.StatusOK},
		{"/apps/shiny-product/app/sub", spaContent, http.StatusOK},
		{"/apps/shiny-product/api/", errorPageContent, http.StatusOK},
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
