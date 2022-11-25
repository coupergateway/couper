package command

import (
	"context"
	"crypto/tls"
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

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server"
)

func TestNewRun(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, hook := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	defaultSettings := config.NewDefaultSettings()

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
			resultSettings := make(chan *config.Settings, 1)
			listenCh := make(chan struct{})
			RunCmdTestCallback = func() {
				close(listenCh)
			}
			RunCmdConfigTestCallback = func(s *config.Settings) {
				resultSettings <- s
				close(resultSettings)
			}

			ctx, shutdown := context.WithCancel(context.Background())
			defer func() {
				n := time.Now()
				shutdown()
				test.WaitForClosedPort(8080)
				subT.Log("shutdown duration: " + time.Since(n).String())
				RunCmdTestCallback = nil
				RunCmdConfigTestCallback = nil
			}()

			runCmd := NewRun(ctx)
			if runCmd == nil {
				subT.Error("create run cmd failed")
				return
			}

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file), "")
			if err != nil {
				subT.Error(err)
			}

			if len(tt.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tt.envs
				})
				defer env.SetTestOsEnviron(os.Environ)
			}

			go func() {
				execErr := runCmd.Execute(tt.args, couperFile, log.WithContext(ctx))
				if execErr != nil {
					subT.Error(execErr)
				}
			}()
			<-listenCh

			result := <-resultSettings
			if !reflect.DeepEqual(result, tt.settings) {
				subT.Errorf("Settings differ: %s:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.settings, result)
			}

			hook.Reset()

			res, err := test.NewHTTPClient().Get("http://localhost:" + strconv.Itoa(result.DefaultPort) + result.HealthPath)
			if err != nil {
				subT.Error(err)
			}

			if res.StatusCode != http.StatusOK {
				subT.Errorf("expected OK, got: %d", res.StatusCode)
			}

			uid := hook.LastEntry().Data["uid"].(string)
			xidLen := len(xid.New().String())
			if result.RequestIDFormat == "uuid4" {
				if len(uid) <= xidLen {
					subT.Errorf("expected uuid4 format, got: %s", uid)
				}
			} else if len(uid) > xidLen {
				subT.Errorf("expected common id format, got: %s", uid)
			}
		})
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

			resultSettings := make(chan *config.Settings, 1)
			RunCmdConfigTestCallback = func(s *config.Settings) {
				resultSettings <- s
				close(resultSettings)
			}
			defer func() { RunCmdConfigTestCallback = nil }()

			runCmd := NewRun(caseCtx)
			if runCmd == nil {
				t.Error("create run cmd failed")
				return
			}

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tc.file), "")
			if err != nil {
				subT.Error(err)
			}

			if len(tc.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tc.envs
				})
				defer env.SetTestOsEnviron(os.Environ)
			}

			go func(asyncT *testing.T) {
				err = runCmd.Execute(tc.args, couperFile, log.WithContext(caseCtx))
				if err != nil {
					asyncT.Error(err)
				}
			}(subT)
			test.WaitForOpenPort(8080)

			settings := <-resultSettings
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

	port := couperFile.Settings.DefaultPort

	// ensure the previous tests aren't listening
	test.WaitForClosedPort(port)
	go func() {
		execErr := runCmd.Execute(Args{"-ca-file=" + tmpFile.Name()}, couperFile, log.WithContext(ctx))
		if execErr != nil {
			helper.Must(execErr)
		}
	}()
	test.WaitForOpenPort(port)

	client := test.NewHTTPClient()

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)

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

	port := couperFile.Settings.DefaultPort

	ctx, shutdown := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer shutdown()

	runCmd := NewRun(ctx)
	if runCmd == nil {
		t.Error("create run cmd failed")
		return
	}

	log, _ := test.NewLogger()

	// ensure the previous tests aren't listening
	test.WaitForClosedPort(port)

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

	_, err = readCertificateFile(malformedFile.Name())
	if err == nil || err.Error() != "error parsing pem ca-certificate: missing pem block" {
		t.Error("expected: error parsing pem ca-certificate: missing pem block")
	}
}
