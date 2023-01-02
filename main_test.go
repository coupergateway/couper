package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config/env"
)

func Test_realmain(t *testing.T) {
	localHook := &logrustest.Hook{}
	testHook = localHook

	base := "server/testdata/settings"

	tests := []struct {
		name    string
		args    []string
		envs    []string
		wantLog string
		want    int
	}{
		{"verify", []string{"couper", "verify", "-f", base + "/10_couper.hcl"}, nil, `10_couper.hcl:2,3-6: Unsupported block type; Blocks of type \"foo\" are not expected here.`, 1},
		{"verify w/o server", []string{"couper", "verify", "-f", base + "/11_couper.hcl"}, nil, `configuration error: missing 'server' block"`, 1},
		{"verify unique map-attr keys", []string{"couper", "verify", "-f", base + "/12_couper.hcl"}, nil, `12_couper.hcl:5,28-8,6: key in an attribute must be unique: 'test-key'; Key must be unique for test-key.`, 1},
		{"common log format & info log level /wo file", []string{"couper", "run"}, nil, `level=error msg="stat %s/couper.hcl: no such file or directory" build=dev`, 1},
		{"common log format via env /wo file", []string{"couper", "run", "-log-format", "json"}, []string{"COUPER_LOG_FORMAT=common"}, `level=error msg="stat %s/couper.hcl: no such file or directory" build=dev`, 1},
		{"info log level via env /wo file", []string{"couper", "run", "-log-level", "debug"}, []string{"COUPER_LOG_LEVEL=info"}, `level=error msg="stat %s/couper.hcl: no such file or directory" build=dev`, 1},
		{"json log format /wo file", []string{"couper", "run", "-log-format", "json"}, nil, `{"build":"dev","level":"error","message":"stat %s/couper.hcl: no such file or directory"`, 1},
		{"json log format via env /wo file", []string{"couper", "run"}, []string{"COUPER_LOG_FORMAT=json"}, `{"build":"dev","level":"error","message":"stat %s/couper.hcl: no such file or directory"`, 1},
		{"non-existent log level /wo file", []string{"couper", "run", "-log-level", "test"}, nil, `level=error msg="stat %s/couper.hcl: no such file or directory" build=dev`, 1},
		{"non-existent log level via env /wo file", []string{"couper", "run"}, []string{"COUPER_LOG_LEVEL=test"}, `level=error msg="stat %s/couper.hcl: no such file or directory" build=dev`, 1},
		{"common log format & info log level /w file", []string{"couper", "run", "-f", base + "/log_default.hcl"}, nil, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		{"common log format via env /w file", []string{"couper", "run", "-f", base + "/log_altered.hcl"}, []string{"COUPER_LOG_FORMAT=common"}, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		{"info log level via env /w file", []string{"couper", "run", "-f", base + "/log_default.hcl"}, []string{"COUPER_LOG_LEVEL=info"}, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		// TODO: format from file currently not possible due to the server error
		{"json log format via env /w file", []string{"couper", "run", "-f", base + "/log_default.hcl"}, []string{"COUPER_LOG_FORMAT=json"}, `{"build":"dev","level":"error","message":"configuration error: missing 'server' block"`, 1},
		{"non-existent log level via env /w file", []string{"couper", "run", "-f", base + "/log_altered.hcl"}, []string{"COUPER_LOG_LEVEL=test"}, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		{"-f w/o file", []string{"couper", "run", "-f"}, nil, `level=error msg="flag needs an argument: -f" build=dev`, 1},
		{"path from env", []string{"couper", "run", "-f", base + "/path_from_env.hcl"}, nil, `level=error msg="configuration error: token: jwt key: read error: open %s/public.pem: no such file or directory" build=dev`, 1},
		{"path from env /w missing key", []string{"couper", "run", "-f", "public/couper.hcl", "-f", base + "/no_key_from_env.hcl"}, nil, "", 0},
		{"empty string in allowed_methods in endpoint", []string{"couper", "run", "-f", base + "/13_couper.hcl"}, nil, `level=error msg="%s/13_couper.hcl:3,5-27: method contains invalid character(s); " build=dev`, 1},
		{"invalid method in allowed_methods in endpoint", []string{"couper", "run", "-f", base + "/14_couper.hcl"}, nil, `level=error msg="%s/14_couper.hcl:3,5-35: method contains invalid character(s); " build=dev`, 1},
		{"invalid method in allowed_methods in api", []string{"couper", "run", "-f", base + "/15_couper.hcl"}, nil, `level=error msg="%s/15_couper.hcl:3,5-35: method contains invalid character(s); " build=dev`, 1},
		{"rate_limit block in anonymous backend", []string{"couper", "run", "-f", base + "/17_couper.hcl"}, nil, `level=error msg="configuration error: anonymous_3_11: anonymous backend 'anonymous_3_11' cannot define 'beta_rate_limit' block(s)" build=dev`, 1},
		{"non-string proxy reference", []string{"couper", "run", "-f", base + "/19_couper.hcl"}, nil, `level=error msg="%s/19_couper.hcl:3,13-14: proxy must evaluate to string; " build=dev`, 1},
		{"proxy reference does not exist", []string{"couper", "run", "-f", base + "/20_couper.hcl"}, nil, `level=error msg="%s/20_couper.hcl:3,14-17: proxy reference is not defined; " build=dev`, 1},
		{"circular backend references", []string{"couper", "run", "-f", base + "/21_couper.hcl"}, nil, `level=error msg="configuration error: <nil>: configuration error; circular reference:`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			if len(tt.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tt.envs
				})
			}

			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(time.Second, cancel)

			if got := realmain(ctx, tt.args); got != tt.want {
				subT.Errorf("realmain() = %v, want %v", got, tt.want)
			}
			env.OsEnviron = os.Environ

			currWD, _ := os.Getwd()

			entry := localHook.LastEntry()
			if entry == nil {
				subT.Error("missing log entry")
				return
			}
			entryStr, _ := entry.String()
			if tt.wantLog == "" && subT.Failed() {
				println(entryStr)
			}

			wantLog := tt.wantLog
			if strings.Contains(wantLog, `msg="%s/`) || strings.Contains(wantLog, `open %s/`) {
				wantLog = fmt.Sprintf(wantLog, filepath.Join(currWD, base))
			} else if strings.Contains(wantLog, `stat %s/`) {
				wantLog = fmt.Sprintf(wantLog, currWD)
			}

			if wantLog != "" && !strings.Contains(entryStr, wantLog) {
				if strings.Contains(wantLog, `failed to load configuration:`) {
					re := regexp.MustCompile(wantLog)

					if !re.MatchString(entryStr) {
						subT.Errorf("\nwant:\t%s\ngot:\t%s\n", wantLog, entryStr)
					}
				} else {
					subT.Errorf("\nwant:\t%s\ngot:\t%s\n", wantLog, entryStr)
				}
			}
		})
	}
}
