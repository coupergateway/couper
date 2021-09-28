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
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config/configload"
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

	conf, err := configload.LoadBytes(confBytes.Bytes(), "conf_test.hcl")
	helper.Must(err)
	conf.Settings.DefaultPort = 0

	srvConf, err := runtime.NewServerConfiguration(conf, log.WithContext(context.TODO()), nil)
	helper.Must(err)

	spaContent, err := os.ReadFile(conf.Servers[0].Spa.BootstrapFile)
	helper.Must(err)

	port := runtime.Port(conf.Settings.DefaultPort)
	gw := server.New(ctx, conf.Context, log.WithContext(ctx), conf.Settings, &runtime.DefaultTimings, port, srvConf[port])
	gw.Listen()
	defer gw.Close()

	connectClient := http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp4", gw.Addr())
		},
		DisableCompression: true,
	}}

	for i, testCase := range []struct {
		path           string
		expectedBody   []byte
		expectedStatus int
	}{
		{"/", []byte("<html><body><h1>route not found error: My custom error template</h1></body></html>"), http.StatusNotFound},
		{"/apps/", []byte("<html><body><h1>route not found error: My custom error template</h1></body></html>"), http.StatusNotFound},
		{"/apps/shiny-product/", []byte("<html><body><h1>route not found error: My custom error template</h1></body></html>"), http.StatusNotFound},
		{"/apps/shiny-product/assets/", []byte("<html><body><h1>route not found error: My custom error template</h1></body></html>"), http.StatusNotFound},
		{"/apps/shiny-product/app/", spaContent, http.StatusOK},
		{"/apps/shiny-product/app/sub", spaContent, http.StatusOK},
		{"/apps/shiny-product/api/", nil, http.StatusNoContent},
		{"/apps/shiny-product/api/foo%20bar:%22baz%22", []byte(`{"message": "route not found error" }`), 404},
	} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com:%s%s", port, testCase.path), nil)
		helper.Must(err)

		res, err := connectClient.Do(req)
		helper.Must(err)

		if res.StatusCode != testCase.expectedStatus {
			t.Errorf("%.2d: expected status %d, got %d", i+1, testCase.expectedStatus, res.StatusCode)
		}

		result, err := io.ReadAll(res.Body)
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
		"origin": "http://" + originBackend.Listener.Addr().String(),
	})
	helper.Must(err)

	log, _ := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := configload.LoadBytes(confBytes.Bytes(), "conf_fileserving.hcl")
	helper.Must(err)

	error404Content := []byte("<html><body><h1>route not found error: My custom error template</h1></body></html>")
	spaContent, err := os.ReadFile(conf.Servers[0].Spa.BootstrapFile)
	helper.Must(err)

	srvConf, err := runtime.NewServerConfiguration(conf, log.WithContext(context.TODO()), nil)
	helper.Must(err)

	couper := server.New(ctx, conf.Context, log.WithContext(ctx), conf.Settings, &runtime.DefaultTimings, runtime.Port(0), srvConf[0])
	couper.Listen()
	defer couper.Close()

	connectClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp4", couper.Addr())
			},
			DisableCompression: true,
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
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", couper.Addr(), testCase.path), nil)
		helper.Must(err)
		req.Host = "example.com"

		res, err := connectClient.Do(req)
		helper.Must(err)

		if res.StatusCode != testCase.expectedStatus {
			t.Fatalf("%.2d: expected status for path %q %d, got %d", i+1, testCase.path, testCase.expectedStatus, res.StatusCode)
		}

		result, err := io.ReadAll(res.Body)
		helper.Must(err)
		helper.Must(res.Body.Close())

		if !bytes.Contains(result, testCase.expectedBody) {
			t.Errorf("%.2d: expected body for path %q:\n%s\ngot:\n%s", i+1, testCase.path, string(testCase.expectedBody), string(result))
		}
	}
	helper.Must(os.Chdir(currentDir)) // defer for error cases, would be to late for normal exit
}

func TestHTTPServer_UUID_Common(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	confPath := "testdata/settings/02_couper.hcl"
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()

	logHook.Reset()
	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(err)

	_, err = client.Do(req)
	helper.Must(err)

	// Wait for log
	time.Sleep(300 * time.Millisecond)

	e := logHook.LastEntry()
	if e == nil {
		t.Fatalf("Missing log line")
	}

	regexCheck := regexp.MustCompile(`^[0-9a-v]{20}$`)
	if !regexCheck.MatchString(e.Data["uid"].(string)) {
		t.Errorf("Expected a common uid format, got %#v", e.Data["uid"])
	}
}

func TestHTTPServer_UUID_uuid4(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	confPath := "testdata/settings/03_couper.hcl"
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()

	logHook.Reset()
	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(err)

	_, err = client.Do(req)
	helper.Must(err)

	// Wait for log
	time.Sleep(300 * time.Millisecond)

	e := logHook.LastEntry()
	if e == nil {
		t.Fatalf("Missing log line")
	}

	regexCheck := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !regexCheck.MatchString(e.Data["uid"].(string)) {
		t.Errorf("Expected a uuid4 uid format, got %#v", e.Data["uid"])
	}
}

func TestHTTPServer_ServeProxyAbortHandler(t *testing.T) {
	configFile := `
server "zipzip" {
	endpoint "/**" {
		proxy {
			backend {
				origin = "%s"
			}
		}
	}
}
`
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Encoding", "gzip")
		gzw := gzip.NewWriter(rw)
		defer func() {
			if r.Header.Get("x-close") != "" {
				return // triggers reverseproxy copyBuffer panic due to missing gzip footer
			}
			if e := gzw.Close(); e != nil {
				t.Error(e)
			}
		}()

		_, err := gzw.Write([]byte(configFile))
		helper.Must(err)

		err = gzw.Flush() // explicit flush, just the gzip footer is missing
		helper.Must(err)
	}))

	shutdown, loghook := newCouperWithBytes([]byte(fmt.Sprintf(configFile, origin.URL)), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	helper.Must(err)

	res, err := newClient().Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected OK, got: %s", res.Status)
		for _, entry := range loghook.AllEntries() {
			t.Log(entry.String())
		}
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)
	helper.Must(res.Body.Close())

	if string(b) != configFile {
		t.Error("Expected same content")
	}

	loghook.Reset()

	// Trigger panic
	req.Header.Set("x-close", "dont")
	res, err = newClient().Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected status %d, got: %d", http.StatusBadGateway, res.StatusCode)
		for _, entry := range loghook.AllEntries() {
			t.Log(entry.String())
		}
	}
}

func TestHTTPServer_Errors(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	confPath := "testdata/settings/03_couper.hcl"
	shutdown, logHook := newCouper(confPath, test.New(t))
	defer shutdown()

	logHook.Reset()
	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(err)

	req.Host = "foo::"
	_, err = client.Do(req)
	helper.Must(err)

	// Wait for log
	time.Sleep(300 * time.Millisecond)

	e := logHook.LastEntry()
	if e == nil {
		t.Fatalf("Missing log line")
	}
}

func TestHTTPServer_RequestID(t *testing.T) {
	client := newClient()

	const (
		confPath = "testdata/settings/"
		validUID = "0123456789-abc+DEF=@/-"
	)

	type expectation struct {
		Headers http.Header
	}

	type testCase struct {
		file         string
		uid          string
		status       int
		expToClient  expectation
		expToBackend expectation
	}

	for i, tc := range []testCase{
		{"07_couper.hcl", "", http.StatusOK,
			expectation{
				Headers: http.Header{
					"Couper-Client-Request-Id": []string{"{{system-id}}"},
				},
			},
			expectation{
				Headers: http.Header{
					"Couper-Backend-Request-Id": []string{"{{system-id}}"},
				},
			},
		},
		{"07_couper.hcl", "XXX", http.StatusBadRequest,
			expectation{
				Headers: http.Header{
					"Couper-Client-Request-Id": []string{"{{system-id}}"},
					"Couper-Error":             []string{"client request error"},
				},
			},
			expectation{},
		},
		{"07_couper.hcl", validUID, http.StatusOK,
			expectation{
				Headers: http.Header{
					"Couper-Client-Request-Id": []string{validUID},
				},
			},
			expectation{
				Headers: http.Header{
					"Client-Request-Id":         []string{validUID},
					"Couper-Backend-Request-Id": []string{validUID},
				},
			},
		},
		{"08_couper.hcl", validUID, http.StatusOK,
			expectation{
				Headers: http.Header{
					"Couper-Request-Id": []string{validUID},
				},
			},
			expectation{
				Headers: http.Header{
					"Client-Request-Id":   []string{validUID},
					"Couper-Request-Id":   []string{validUID},
					"Request-Id-From-Var": []string{validUID},
				},
			},
		},
		{"08_couper.hcl", "", http.StatusOK,
			expectation{
				Headers: http.Header{
					"Couper-Request-Id": []string{"{{system-id}}"},
				},
			},
			expectation{
				Headers: http.Header{
					"Couper-Request-Id":   []string{"{{system-id}}"},
					"Request-Id-From-Var": []string{"{{system-id}}"},
				},
			},
		},
		{"09_couper.hcl", validUID, http.StatusOK,
			expectation{
				Headers: http.Header{},
			},
			expectation{
				Headers: http.Header{
					"Client-Request-Id":   []string{validUID},
					"Request-ID-From-Var": []string{validUID},
				},
			},
		},
	} {
		t.Run("_"+tc.file, func(subT *testing.T) {
			helper := test.New(subT)
			shutdown, hook := newCouper(path.Join(confPath, tc.file), helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080", nil)
			helper.Must(err)

			if tc.uid != "" {
				req.Header.Set("Client-Request-ID", tc.uid)
			}

			hook.Reset()
			res, err := client.Do(req)
			helper.Must(err)

			// Wait for log
			time.Sleep(300 * time.Millisecond)

			lastLog := hook.LastEntry()

			getHeaderValue := func(header http.Header, name string) string {
				return strings.Replace(
					header.Get(name),
					"{{system-id}}",
					lastLog.Data["uid"].(string),
					-1,
				)
			}

			if tc.status != res.StatusCode {
				subT.Errorf("Unexpected status code given: %d", res.StatusCode)
				return
			}

			if tc.status == http.StatusOK {
				if lastLog.Message != "" {
					subT.Errorf("Unexpected log message given: %s", lastLog.Message)
				}

				for k := range tc.expToClient.Headers {
					v := getHeaderValue(tc.expToClient.Headers, k)

					if v != res.Header.Get(k) {
						subT.Errorf("%d: Unexpected response header %q sent: %s, want: %q", i, k, res.Header.Get(k), v)
					}
				}

				body, err := io.ReadAll(res.Body)
				helper.Must(err)
				helper.Must(res.Body.Close())

				var jsonResult expectation
				err = json.Unmarshal(body, &jsonResult)
				if err != nil {
					t.Errorf("unmarshal json: %v: got:\n%s", err, string(body))
				}

				for k := range tc.expToBackend.Headers {
					v := getHeaderValue(tc.expToBackend.Headers, k)

					if v != jsonResult.Headers.Get(k) {
						subT.Errorf("%d: Unexpected header %q sent to backend: %q, want: %q", i, k, jsonResult.Headers.Get(k), v)
					}
				}
			} else {
				exp := fmt.Sprintf("client request error: invalid request-id header value: Client-Request-ID: %s", tc.uid)
				if lastLog.Message != exp {
					subT.Errorf("\nWant:\t%s\nGot:\t%s", exp, lastLog.Message)
				}

				for k := range tc.expToClient.Headers {
					v := getHeaderValue(tc.expToClient.Headers, k)

					if v != res.Header.Get(k) {
						subT.Errorf("Unexpected response header %q: %q, want: %q", k, res.Header.Get(k), v)
					}
				}
			}
		})
	}
}
