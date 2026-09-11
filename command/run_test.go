package command

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/xid"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/env"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server"
)

func TestNewRun(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, _ := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	defaultSettings := config.NewDefaultSettings()
	defaultSettings.BindAddresses = make(map[string]string)
	defaultSettings.BindAddresses[""] = "tcp"
	defaultSettings.BindAddress = "*"

	tests := []struct {
		name     string
		file     string
		args     Args
		envs     []string
		settings *config.Settings
	}{
		{"defaults from file", "01_defaults.hcl", nil, nil, defaultSettings},
		{"overrides from file", "02_changed_defaults.hcl", nil, nil, &config.Settings{
			AcceptForwarded:          &config.AcceptForwarded{},
			BindAddress:              "*",
			BindAddresses:            map[string]string{"": "tcp"},
			DefaultPort:              9090,
			HealthPath:               "/status/health",
			LogFormat:                defaultSettings.LogFormat,
			LogLevel:                 defaultSettings.LogLevel,
			PProfPort:                defaultSettings.PProfPort,
			NoProxyFromEnv:           true,
			RequestIDBackendHeader:   defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:    defaultSettings.RequestIDClientHeader,
			RequestIDFormat:          "uuid4",
			TelemetryMetricsEndpoint: defaultSettings.TelemetryMetricsEndpoint,
			TelemetryMetricsExporter: defaultSettings.TelemetryMetricsExporter,
			TelemetryMetricsPort:     defaultSettings.TelemetryMetricsPort,
			TelemetryServiceName:     "couper",
			TelemetryTracesEndpoint:  defaultSettings.TelemetryTracesEndpoint,
			XForwardedHost:           true,
		}},
		{"defaults with flag port", "01_defaults.hcl", Args{"-p", "9876"}, nil, &config.Settings{
			AcceptForwarded:          &config.AcceptForwarded{},
			BindAddress:              "*",
			BindAddresses:            map[string]string{"": "tcp"},
			DefaultPort:              9876,
			HealthPath:               defaultSettings.HealthPath,
			LogFormat:                defaultSettings.LogFormat,
			LogLevel:                 defaultSettings.LogLevel,
			PProfPort:                defaultSettings.PProfPort,
			RequestIDBackendHeader:   defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:    defaultSettings.RequestIDClientHeader,
			RequestIDFormat:          defaultSettings.LogFormat,
			TelemetryMetricsEndpoint: defaultSettings.TelemetryMetricsEndpoint,
			TelemetryMetricsExporter: defaultSettings.TelemetryMetricsExporter,
			TelemetryMetricsPort:     defaultSettings.TelemetryMetricsPort,
			TelemetryServiceName:     "couper",
			TelemetryTracesEndpoint:  defaultSettings.TelemetryTracesEndpoint,
		}},
		{"defaults with flag and env port", "01_defaults.hcl", Args{"-p", "9876"}, []string{"COUPER_DEFAULT_PORT=4561"}, &config.Settings{
			AcceptForwarded:          &config.AcceptForwarded{},
			BindAddress:              "*",
			BindAddresses:            map[string]string{"": "tcp"},
			DefaultPort:              4561,
			HealthPath:               defaultSettings.HealthPath,
			LogFormat:                defaultSettings.LogFormat,
			LogLevel:                 defaultSettings.LogLevel,
			PProfPort:                defaultSettings.PProfPort,
			RequestIDBackendHeader:   defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:    defaultSettings.RequestIDClientHeader,
			RequestIDFormat:          defaultSettings.LogFormat,
			TelemetryMetricsEndpoint: defaultSettings.TelemetryMetricsEndpoint,
			TelemetryMetricsExporter: defaultSettings.TelemetryMetricsExporter,
			TelemetryMetricsPort:     defaultSettings.TelemetryMetricsPort,
			TelemetryServiceName:     "couper",
			TelemetryTracesEndpoint:  defaultSettings.TelemetryTracesEndpoint,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file), "")
			if err != nil {
				subT.Fatal(err)
			}

			if len(tt.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tt.envs
				})
				defer env.SetTestOsEnviron(os.Environ)
			}

			runCmd := NewRun(ctx)
			if runCmd == nil {
				subT.Fatal("create run cmd failed")
			}

			if err = runCmd.applySettings(tt.args, couperFile, log.WithContext(ctx)); err != nil {
				subT.Fatal(err)
			}

			if !reflect.DeepEqual(couperFile.Settings, tt.settings) {
				subT.Errorf("Settings differ: %s:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.settings, couperFile.Settings)
			}
		})
	}
}

func TestNewRun_Serve(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, hook := logrustest.NewNullLogger()

	tests := []struct {
		name      string
		file      string
		wantUUID4 bool
	}{
		{"common request id format", "01_defaults.hcl", false},
		{"uuid4 request id format", "02_changed_defaults.hcl", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			resultSettings := make(chan *config.Settings, 1)
			listenCh := make(chan []string, 1)
			execErrCh := make(chan error, 1)

			RunCmdTestCallback = func(listenPorts []string) {
				listenCh <- listenPorts
			}
			RunCmdConfigTestCallback = func(s *config.Settings) {
				resultSettings <- s
			}

			ctx, shutdown := context.WithTimeout(context.Background(), 22*time.Second)
			defer func() {
				shutdown()
				RunCmdTestCallback = nil
				RunCmdConfigTestCallback = nil
			}()

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file), "")
			if err != nil {
				subT.Fatal(err)
			}

			runCmd := NewRun(ctx)
			if runCmd == nil {
				subT.Fatal("create run cmd failed")
			}

			go func() {
				execErrCh <- runCmd.Execute(Args{"-p", "0"}, couperFile, log.WithContext(ctx))
			}()

			var listenPorts []string
			select {
			case listenPorts = <-listenCh:
			case execErr := <-execErrCh:
				subT.Fatalf("couper did not start to listen: %v", execErr)
			case <-ctx.Done():
				subT.Fatal("timeout while waiting for couper to listen")
			}

			if len(listenPorts) != 1 {
				subT.Fatalf("want one listen port, got: %v", listenPorts)
			}

			if port, atoiErr := strconv.Atoi(listenPorts[0]); atoiErr != nil || port <= 0 {
				subT.Fatalf("want an assigned port, got: %q", listenPorts[0])
			}

			settings := <-resultSettings
			hook.Reset()

			res, err := test.NewHTTPClient().Get("http://localhost:" + listenPorts[0] + settings.HealthPath)
			if err != nil {
				subT.Fatal(err)
			}

			if res.StatusCode != http.StatusOK {
				subT.Errorf("expected OK, got: %d", res.StatusCode)
			}

			uid, _ := hook.LastEntry().Data["uid"].(string)
			xidLen := len(xid.New().String())
			if tt.wantUUID4 {
				if len(uid) <= xidLen {
					subT.Errorf("expected uuid4 format, got: %s", uid)
				}
			} else if len(uid) > xidLen {
				subT.Errorf("expected common id format, got: %s", uid)
			}

			shutdown()
			select {
			case execErr := <-execErrCh:
				if execErr != nil {
					subT.Errorf("run returned an error: %v", execErr)
				}
			case <-time.After(5 * time.Second):
				subT.Error("run did not return after the shutdown")
			}
		})
	}
}

func TestNewRun_ListenError(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, _ := logrustest.NewNullLogger()

	// A wildcard bind coexists with one to a specific address, so hold the wildcard
	// Couper binds by default.
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", "01_defaults.hcl"), "")
	if err != nil {
		t.Fatal(err)
	}

	listenCh := make(chan []string, 1)
	RunCmdTestCallback = func(listenPorts []string) {
		listenCh <- listenPorts
	}
	defer func() { RunCmdTestCallback = nil }()

	ctx, shutdown := context.WithTimeout(context.Background(), 22*time.Second)
	defer shutdown()

	execErrCh := make(chan error, 1)
	go func() {
		execErrCh <- NewRun(ctx).Execute(Args{"-p", port}, couperFile, log.WithContext(ctx))
	}()

	select {
	case execErr := <-execErrCh:
		if execErr == nil {
			t.Fatal("expected a listen error")
		}
		if !strings.Contains(execErr.Error(), "address already in use") {
			t.Errorf("want an address-in-use error, got: %v", execErr)
		}
	case listenPorts := <-listenCh:
		t.Fatalf("expected no listener, got: %v", listenPorts)
	case <-ctx.Done():
		t.Fatal("Execute did not return the listen error")
	}
}

func TestAcceptForwarded(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, _ := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	tests := []struct {
		name     string
		file     string
		args     Args
		envs     []string
		expProto bool
		expHost  bool
		expPort  bool
	}{
		{"defaults", "01_defaults.hcl", nil, nil, false, false, false},
		{"accept by settings", "03_accept.hcl", nil, nil, true, true, true},
		{"accept by option", "01_defaults.hcl", Args{"-accept-forwarded-url", "proto,host,port"}, nil, true, true, true},
		{"accept by env", "01_defaults.hcl", nil, []string{"COUPER_ACCEPT_FORWARDED_URL=proto,host,port"}, true, true, true},
	}

	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	for _, testcase := range tests {
		t.Run(testcase.name, func(subT *testing.T) {
			tc := testcase

			caseCtx, caseCancel := context.WithCancel(ctx)
			defer caseCancel()

			runCmd := NewRun(caseCtx)
			if runCmd == nil {
				subT.Fatal("create run cmd failed")
			}

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tc.file), "")
			if err != nil {
				subT.Fatal(err)
			}

			if len(tc.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tc.envs
				})
				defer env.SetTestOsEnviron(os.Environ)
			}

			if err = runCmd.applySettings(tc.args, couperFile, log.WithContext(caseCtx)); err != nil {
				subT.Fatal(err)
			}

			settings := couperFile.Settings
			if settings.AcceptsForwardedProtocol() != tc.expProto {
				subT.Errorf("%s: AcceptsForwardedProtocol() differ:\nwant:\t%#v\ngot:\t%#v\n", tc.name, tc.expProto, settings.AcceptsForwardedProtocol())
			}
			if settings.AcceptsForwardedHost() != tc.expHost {
				subT.Errorf("%s: AcceptsForwardedHost() differ:\nwant:\t%#v\ngot:\t%#v\n", tc.name, tc.expHost, settings.AcceptsForwardedHost())
			}
			if settings.AcceptsForwardedPort() != tc.expPort {
				subT.Errorf("%s: AcceptsForwardedPort() differ:\nwant:\t%#v\ngot:\t%#v\n", tc.name, tc.expPort, settings.AcceptsForwardedPort())
			}
		})
	}
}

func TestArgs_CAFile(t *testing.T) {
	helper := test.New(t)

	log, hook := test.NewLogger()
	defer func() {
		if t.Failed() {
			for _, entry := range hook.AllEntries() {
				t.Log(entry.String())
			}
		}
	}()

	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	runCmd := NewRun(ctx)
	if runCmd == nil {
		t.Error("create run cmd failed")
		return
	}

	expiresIn := time.Minute
	selfSigned, err := server.NewCertificate(expiresIn, nil, nil)
	helper.Must(err)

	tmpFile, err := os.CreateTemp("", "ca.cert")
	helper.Must(err)
	_, err = tmpFile.Write(selfSigned.CACertificate.Certificate)
	helper.Must(err)
	helper.Must(tmpFile.Close())
	defer os.Remove(tmpFile.Name())

	var healthCheckSeen uint32

	srv := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.Header.Get("User-Agent"), "health-check") {
			atomic.StoreUint32(&healthCheckSeen, 1)
		}
		writer.WriteHeader(http.StatusNoContent)

		// force close to trigger a new handshake
		hj, ok := writer.(http.Hijacker)
		if !ok {
			t.Error("expected hijacker")
		}

		conn, _, herr := hj.Hijack()
		if herr != nil {
			t.Error(herr)
		}

		conn.Close()
	}))
	defer srv.Close()

	srv.TLS.Certificates = []tls.Certificate{*selfSigned.Server}

	couperHCL := `server {
	endpoint "/" {
		request {
			backend = "tls"
		}
	}
}

definitions {
	backend "tls" {
		origin = "` + srv.URL + `"
		beta_health {
			failure_threshold = 0
		}
	}
}`

	couperFile, err := configload.LoadBytes([]byte(couperHCL), "ca-file-test.hcl")
	helper.Must(err)

	listenCh := make(chan []string, 1)
	RunCmdTestCallback = func(listenPorts []string) {
		listenCh <- listenPorts
	}
	defer func() { RunCmdTestCallback = nil }()

	execErrCh := make(chan error, 1)
	go func() {
		execErrCh <- runCmd.Execute(Args{"-ca-file=" + tmpFile.Name(), "-p", "0"}, couperFile, log.WithContext(ctx))
	}()
	defer func() {
		shutdown()
		select {
		case execErr := <-execErrCh:
			if execErr != nil {
				t.Errorf("run returned an error: %v", execErr)
			}
		case <-time.After(5 * time.Second):
			t.Error("run did not return after the shutdown")
		}
	}()

	var port string
	select {
	case listenPorts := <-listenCh:
		if len(listenPorts) != 1 {
			t.Fatalf("want one listen port, got: %v", listenPorts)
		}
		port = listenPorts[0]
	case execErr := <-execErrCh:
		t.Fatalf("couper did not start to listen: %v", execErr)
	case <-ctx.Done():
		t.Fatal("timeout while waiting for couper to listen")
	}

	client := test.NewHTTPClient()

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:"+port+"/", nil)

	// ca before
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusNoContent {
		t.Error("unexpected status code")
	}

	if atomic.LoadUint32(&healthCheckSeen) != 1 {
		t.Error("expected a successful tls health check")
	}
}

func TestCAFile_Run(t *testing.T) {
	helper := test.New(t)

	couperHCL := `server {}
settings {
  ca_file = "/tmp/not-there.pem"
}
`

	couperFile, err := configload.LoadBytes([]byte(couperHCL), "ca-file-test.hcl")
	helper.Must(err)

	ctx, shutdown := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer shutdown()

	runCmd := NewRun(ctx)
	if runCmd == nil {
		t.Error("create run cmd failed")
		return
	}

	log, _ := test.NewLogger()

	execErr := runCmd.Execute(Args{}, couperFile, log.WithContext(ctx))
	if execErr == nil {
		t.Error("expected a ca read error")
	} else {
		want := "error reading ca-certificate: open /tmp/not-there.pem: no such file or directory"
		if execErr.Error() != want {
			t.Errorf("want: %q, got: %q", want, execErr.Error())
		}
	}
}

func TestReadCAFile(t *testing.T) {
	helper := test.New(t)

	_, err := readCertificateFile("/does/not/exist.cert")
	if err == nil {
		t.Error("expected file error")
	} else if err.Error() != "error reading ca-certificate: open /does/not/exist.cert: no such file or directory" {
		t.Error("expected no such file error")
	}

	tmpFile, err := os.CreateTemp("", "empty.cert")
	helper.Must(err)
	defer os.Remove(tmpFile.Name())

	_, err = readCertificateFile(tmpFile.Name())
	if err == nil {
		t.Error("expected empty file error")
	} else if err.Error() != `error reading ca-certificate: empty file: "`+tmpFile.Name()+`"` {
		t.Error("expected empty file error with file-name")
	}

	malformedFile, err := os.CreateTemp("", "broken.cert")
	helper.Must(err)
	defer os.Remove(malformedFile.Name())

	ssc, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	_, err = malformedFile.Write(ssc.CACertificate.Certificate[:100]) // incomplete
	helper.Must(err)

	exp := "error parsing pem ca-certificate: has no valid X509 certificate"
	_, err = readCertificateFile(malformedFile.Name())
	if err == nil {
		t.Errorf("expected: %s", exp)
	} else if err.Error() != exp {
		t.Errorf("expected: %s\nwas: %s", exp, err.Error())
	}

	okFile, err := os.CreateTemp("", "ok.cert")
	helper.Must(err)
	defer os.Remove(okFile.Name())

	_, err = okFile.Write(ssc.CACertificate.Certificate[:])
	helper.Must(err)
	_, err = okFile.WriteString("bogus trailing stuff")
	helper.Must(err)

	_, err = readCertificateFile(okFile.Name())
	if err != nil {
		t.Error("expected no error")
	}
}
