package main

import (
	"os"
	"strings"
	"testing"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config/env"
)

func Test_realmain(t *testing.T) {
	localHook := &logrustest.Hook{}
	testHook = localHook

	base := "server/testdata/settings"
	wd, _ := os.Getwd()

	tests := []struct {
		name    string
		args    []string
		envs    []string
		wantLog string
		want    int
	}{
		{"common log format & info log level /wo file", []string{"couper", "run"}, nil, `level=error msg="failed to load configuration: open couper.hcl: no such file or directory" build=dev`, 1},
		{"common log format via env /wo file", []string{"couper", "run", "-log-format", "json"}, []string{"COUPER_LOG_FORMAT=common"}, `level=error msg="failed to load configuration: open couper.hcl: no such file or directory" build=dev`, 1},
		{"info log level via env /wo file", []string{"couper", "run", "-log-level", "debug"}, []string{"COUPER_LOG_LEVEL=info"}, `level=error msg="failed to load configuration: open couper.hcl: no such file or directory" build=dev`, 1},
		{"json log format /wo file", []string{"couper", "run", "-log-format", "json"}, nil, `{"build":"dev","level":"error","message":"failed to load configuration: open couper.hcl: no such file or directory"`, 1},
		{"json log format via env /wo file", []string{"couper", "run"}, []string{"COUPER_LOG_FORMAT=json"}, `{"build":"dev","level":"error","message":"failed to load configuration: open couper.hcl: no such file or directory"`, 1},
		{"non-existent log level /wo file", []string{"couper", "run", "-log-level", "test"}, nil, `level=error msg="failed to load configuration: open couper.hcl: no such file or directory" build=dev`, 1},
		{"non-existent log level via env /wo file", []string{"couper", "run"}, []string{"COUPER_LOG_LEVEL=test"}, `level=error msg="failed to load configuration: open couper.hcl: no such file or directory" build=dev`, 1},
		{"common log format & info log level /w file", []string{"couper", "run", "-f", base + "/log_default.hcl"}, nil, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		{"common log format via env /w file", []string{"couper", "run", "-f", base + "/log_altered.hcl"}, []string{"COUPER_LOG_FORMAT=common"}, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		{"info log level via env /w file", []string{"couper", "run", "-f", base + "/log_default.hcl"}, []string{"COUPER_LOG_LEVEL=info"}, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		// TODO: format from file currently not possible due to the server error
		{"json log format via env /w file", []string{"couper", "run", "-f", base + "/log_default.hcl"}, []string{"COUPER_LOG_FORMAT=json"}, `{"build":"dev","level":"error","message":"configuration error: missing 'server' block"`, 1},
		{"non-existent log level via env /w file", []string{"couper", "run", "-f", base + "/log_altered.hcl"}, []string{"COUPER_LOG_LEVEL=test"}, `level=error msg="configuration error: missing 'server' block" build=dev`, 1},
		{"-f w/o file", []string{"couper", "run", "-f"}, nil, `level=error msg="flag needs an argument: -f" build=dev`, 1},
		{"undefined AC", []string{"couper", "run", "-f", base + "/04_couper.hcl"}, nil, `level=error msg="accessControl is not defined: undefined" build=dev`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			if len(tt.envs) > 0 {
				env.SetTestOsEnviron(func() []string {
					return tt.envs
				})
			}

			if got := realmain(tt.args); got != tt.want {
				subT.Errorf("realmain() = %v, want %v", got, tt.want)
			}
			env.OsEnviron = os.Environ

			entry, _ := localHook.LastEntry().String()
			//println(entry)
			if tt.wantLog != "" && !strings.Contains(entry, tt.wantLog) {
				subT.Errorf("\nwant:\t%s\ngot:\t%s\n", tt.wantLog, entry)
			}

			err := os.Chdir(wd)
			if err != nil {
				subT.Error(err)
			}
		})
	}
}
