package server_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
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
		"origin_address": originBackend.Listener.Addr().String(),
		"origin_host":    expectedAPIHost,
	})
	if err != nil {
		t.Fatal(err)
	}

	log, _ := logrustest.NewNullLogger()
	//log.Out = os.Stdout

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

	for _, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", errorPageContent, http.StatusNotFound},
		{"/apps/", errorPageContent, http.StatusNotFound},
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
