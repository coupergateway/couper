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
	"time"

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
			DefaultPort:            9090,
			HealthPath:             "/status/health",
			NoProxyFromEnv:         true,
			LogFormat:              defaultSettings.LogFormat,
			LogLevel:               defaultSettings.LogLevel,
			RequestIDFormat:        "uuid4",
			RequestIDBackendHeader: defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:  defaultSettings.RequestIDClientHeader,
			XForwardedHost:         true,
			AcceptForwardedURL:     []string{},
			AcceptForwarded:        &config.AcceptForwarded{},
		}},
		{"defaults with flag port", "01_defaults.hcl", Args{"-p", "9876"}, nil, &config.Settings{
			DefaultPort:            9876,
			HealthPath:             defaultSettings.HealthPath,
			LogFormat:              defaultSettings.LogFormat,
			LogLevel:               defaultSettings.LogLevel,
			RequestIDFormat:        defaultSettings.LogFormat,
			RequestIDBackendHeader: defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:  defaultSettings.RequestIDClientHeader,
			AcceptForwardedURL:     []string{},
			AcceptForwarded:        &config.AcceptForwarded{},
		}},
		{"defaults with flag and env port", "01_defaults.hcl", Args{"-p", "9876"}, []string{"COUPER_DEFAULT_PORT=4561"}, &config.Settings{
			DefaultPort:            4561,
			HealthPath:             defaultSettings.HealthPath,
			LogFormat:              defaultSettings.LogFormat,
			LogLevel:               defaultSettings.LogLevel,
			RequestIDFormat:        defaultSettings.LogFormat,
			RequestIDBackendHeader: defaultSettings.RequestIDBackendHeader,
			RequestIDClientHeader:  defaultSettings.RequestIDClientHeader,
			AcceptForwardedURL:     []string{},
			AcceptForwarded:        &config.AcceptForwarded{},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(t)
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()

			runCmd := NewRun(ctx)
			if runCmd == nil {
				t.Error("create run cmd failed")
				return
			}

			couperFile, fileErr := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file))
			helper.Must(fileErr)

			if len(tt.envs) > 0 {
				env.OsEnviron = func() []string {
					return tt.envs
				}
				defer func() { env.OsEnviron = os.Environ }()
			}

			go func() {
				helper.Must(runCmd.Execute(tt.args, couperFile, log.WithContext(ctx)))
			}()
			time.Sleep(time.Second / 4)
			runCmd.settingsMu.Lock()
			if !reflect.DeepEqual(couperFile.Settings, tt.settings) {
				t.Errorf("Settings differ:\nwant:\t%#v\ngot:\t%#v\n", tt.settings, couperFile.Settings)
			}
			runCmd.settingsMu.Unlock()

			hook.Reset()

			res, resErr := test.NewHTTPClient().Get("http://localhost:" + strconv.Itoa(couperFile.Settings.DefaultPort) + couperFile.Settings.HealthPath)
			helper.Must(resErr)

			if res.StatusCode != http.StatusOK {
				subT.Errorf("expected OK, got: %d", res.StatusCode)
			}

			uid := hook.LastEntry().Data["uid"].(string)
			xidLen := len(xid.New().String())
			if couperFile.Settings.RequestIDFormat == "uuid4" {
				if len(uid) <= xidLen {
					t.Errorf("expected uuid4 format, got: %s", uid)
				}
			} else if len(uid) > xidLen {
				t.Errorf("expected common id format, got: %s", uid)
			}
		})
		time.Sleep(time.Second / 2) // shutdown
	}
}

func TestAcceptForwarded(t *testing.T) {
	_, currFile, _, _ := runtime.Caller(0)
	wd := filepath.Dir(currFile)

	log, hook := logrustest.NewNullLogger()
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
			helper := test.New(t)
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()

			runCmd := NewRun(ctx)
			if runCmd == nil {
				t.Error("create run cmd failed")
				return
			}

			couperFile, fileErr := configload.LoadFile(filepath.Join(wd, "testdata/settings", tt.file))
			helper.Must(fileErr)

			if len(tt.envs) > 0 {
				env.OsEnviron = func() []string {
					return tt.envs
				}
				defer func() { env.OsEnviron = os.Environ }()
			}

			go func() {
				helper.Must(runCmd.Execute(tt.args, couperFile, log.WithContext(ctx)))
			}()
			time.Sleep(time.Second / 4)
			runCmd.settingsMu.Lock()

			if couperFile.Settings.AcceptsForwardedProtocol() != tt.expProto {
				t.Errorf("%s: AcceptsForwardedProtocol() differ:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.expProto, couperFile.Settings.AcceptsForwardedProtocol())
			}
			if couperFile.Settings.AcceptsForwardedHost() != tt.expHost {
				t.Errorf("%s: AcceptsForwardedHost() differ:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.expHost, couperFile.Settings.AcceptsForwardedHost())
			}
			if couperFile.Settings.AcceptsForwardedPort() != tt.expPort {
				t.Errorf("%s: AcceptsForwardedPort() differ:\nwant:\t%#v\ngot:\t%#v\n", tt.name, tt.expPort, couperFile.Settings.AcceptsForwardedPort())
			}
			runCmd.settingsMu.Unlock()

			hook.Reset()
		})
		time.Sleep(time.Second / 2) // shutdown
	}
}
