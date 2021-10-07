package command

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/rs/xid"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/internal/test"
)

func TestNewRun(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, hook := logrustest.NewNullLogger()
	//log.Out = os.Stdout

	defaultSettings := config.DefaultSettings

	tests := []struct {
		name     string
		file     string
		args     Args
		envs     []string
		settings *config.Settings
	}{
		{"defaults from file", "01_defaults.hcl", nil, nil, &defaultSettings},
		{"overrides from file", "02_changed_defaults.hcl", nil, nil, &config.Settings{
			AcceptForwarded:          &config.AcceptForwarded{},
			AcceptForwardedURL:       []string{},
			DefaultPort:              9090,
			HealthPath:               "/status/health",
			LogFormat:                defaultSettings.LogFormat,
			LogLevel:                 defaultSettings.LogLevel,
			NoProxyFromEnv:           true,
			RequestIDBackendHeader:   defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:    defaultSettings.RequestIDClientHeader,
			RequestIDFormat:          "uuid4",
			TelemetryMetricsEndpoint: defaultSettings.TelemetryMetricsEndpoint,
			TelemetryMetricsExporter: defaultSettings.TelemetryMetricsExporter,
			TelemetryMetricsPort:     defaultSettings.TelemetryMetricsPort,
			TelemetryTracesEndpoint:  defaultSettings.TelemetryTracesEndpoint,
			XForwardedHost:           true,
		}},
		{"defaults with flag port", "01_defaults.hcl", Args{"-p", "9876"}, nil, &config.Settings{
			AcceptForwarded:          &config.AcceptForwarded{},
			AcceptForwardedURL:       []string{},
			DefaultPort:              9876,
			HealthPath:               defaultSettings.HealthPath,
			LogFormat:                defaultSettings.LogFormat,
			LogLevel:                 defaultSettings.LogLevel,
			RequestIDBackendHeader:   defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:    defaultSettings.RequestIDClientHeader,
			RequestIDFormat:          defaultSettings.LogFormat,
			TelemetryMetricsEndpoint: defaultSettings.TelemetryMetricsEndpoint,
			TelemetryMetricsExporter: defaultSettings.TelemetryMetricsExporter,
			TelemetryMetricsPort:     defaultSettings.TelemetryMetricsPort,
			TelemetryTracesEndpoint:  defaultSettings.TelemetryTracesEndpoint,
		}},
		{"defaults with flag and env port", "01_defaults.hcl", Args{"-p", "9876"}, []string{"COUPER_DEFAULT_PORT=4561"}, &config.Settings{
			AcceptForwarded:          &config.AcceptForwarded{},
			AcceptForwardedURL:       []string{},
			DefaultPort:              4561,
			HealthPath:               defaultSettings.HealthPath,
			LogFormat:                defaultSettings.LogFormat,
			LogLevel:                 defaultSettings.LogLevel,
			RequestIDBackendHeader:   defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:    defaultSettings.RequestIDClientHeader,
			RequestIDFormat:          defaultSettings.LogFormat,
			TelemetryMetricsEndpoint: defaultSettings.TelemetryMetricsEndpoint,
			TelemetryMetricsExporter: defaultSettings.TelemetryMetricsExporter,
			TelemetryMetricsPort:     defaultSettings.TelemetryMetricsPort,
			TelemetryTracesEndpoint:  defaultSettings.TelemetryTracesEndpoint,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()

			runCmd := NewRun(ctx)
			if runCmd == nil {
				subT.Error("create run cmd failed")
				return
			}

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file))
			if err != nil {
				subT.Error(err)
			}

			// settings must be locked, so assign port now
			port := tt.settings.DefaultPort

			if len(tt.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tt.envs
				})
				defer env.SetTestOsEnviron(os.Environ)
			}

			// ensure the previous test aren't listening
			test.WaitForClosedPort(port)
			go func() {
				execErr := runCmd.Execute(tt.args, couperFile, log.WithContext(ctx))
				if execErr != nil {
					subT.Error(execErr)
				}
			}()
			test.WaitForOpenPort(port)

			runCmd.settingsMu.Lock()
			if !reflect.DeepEqual(couperFile.Settings, tt.settings) {
				subT.Errorf("Settings differ: %s:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.settings, couperFile.Settings)
			}
			runCmd.settingsMu.Unlock()

			hook.Reset()

			res, err := test.NewHTTPClient().Get("http://localhost:" + strconv.Itoa(couperFile.Settings.DefaultPort) + couperFile.Settings.HealthPath)
			if err != nil {
				subT.Error(err)
			}

			if res.StatusCode != http.StatusOK {
				subT.Errorf("expected OK, got: %d", res.StatusCode)
			}

			uid := hook.LastEntry().Data["uid"].(string)
			xidLen := len(xid.New().String())
			if couperFile.Settings.RequestIDFormat == "uuid4" {
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
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()

			runCmd := NewRun(ctx)
			if runCmd == nil {
				t.Error("create run cmd failed")
				return
			}

			couperFile, err := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file))
			if err != nil {
				subT.Error(err)
			}

			// settings must be locked, so assign port now
			port := couperFile.Settings.DefaultPort

			if len(tt.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tt.envs
				})
				defer env.SetTestOsEnviron(os.Environ)
			}

			// ensure the previous test aren't listening
			test.WaitForClosedPort(port)
			go func() {
				execErr := runCmd.Execute(tt.args, couperFile, log.WithContext(ctx))
				if execErr != nil {
					subT.Error(execErr)
				}
			}()
			test.WaitForOpenPort(port)

			runCmd.settingsMu.Lock()

			if couperFile.Settings.AcceptsForwardedProtocol() != tt.expProto {
				subT.Errorf("%s: AcceptsForwardedProtocol() differ:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.expProto, couperFile.Settings.AcceptsForwardedProtocol())
			}
			if couperFile.Settings.AcceptsForwardedHost() != tt.expHost {
				subT.Errorf("%s: AcceptsForwardedHost() differ:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.expHost, couperFile.Settings.AcceptsForwardedHost())
			}
			if couperFile.Settings.AcceptsForwardedPort() != tt.expPort {
				subT.Errorf("%s: AcceptsForwardedPort() differ:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.expPort, couperFile.Settings.AcceptsForwardedPort())
			}
			runCmd.settingsMu.Unlock()
		})
	}
}
