package runtime_test

import (
	"context"
	goerrors "errors"
	"fmt"
	goruntime "runtime"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/config/runtime"
	couperErr "github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/test"
)

func TestDuplicateEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		endpoints []string
	}{
		{
			"shared API base path: create catch-all",
			`api {
			   access_control = ["a"]
			 }
			 api {
			   access_control = ["a"]
			 }`,
			[]string{"/**"},
		},
		{
			"shared API base path: create catch-all",
			`base_path = "/p"
			 api {
			   access_control = ["a"]
			 }
			 api {
			   access_control = ["a"]
			 }`,
			[]string{"/p/**"},
		},
		{
			"shared API base path: create catch-all",
			`api {}
			 api {
			   access_control = ["a"]
			 }`,
			[]string{"/**"},
		},
		{
			"shared API base path w/o access control: no catch-all",
			`api {
			 }
			 api {
			 }`,
			[]string{},
		},
		{
			"shared API base path: create catch-all",
			`access_control = ["a"]
			 api {}
			 api {}
			 api {
			   base_path = "/p"
			   endpoint "/**" {
			     response {}
		       }
			 }`,
			[]string{"/**", "/p/**"},
		},
		{
			"unique base paths: create catch-all twice",
			`access_control = ["a"]
			 api {
			   base_path = "/"
			 }
			 api {
			   base_path = "/p"
			 }`,
			[]string{"/**", "/p/**"},
		},
		{
			"unique base paths: create catch-all twice",
			`access_control = ["a"]
			 api {
			   base_path = "/p"
			 }
			 api {
			   base_path = "/"
			 }`,
			[]string{"/**", "/p/**"},
		},
		{
			"user defined /** endpoint in 1st API: no extra catch-all",
			`api {
			   endpoint "/**" {
			     access_control = ["a"]
			     response {}
			   }
			 }
			 api {
			   access_control = ["a"]
			 }
			`,
			[]string{"/**"},
		},
		{
			"user defined /** endpoint in 2nd API: no extra catch-all",
			`access_control = ["a"]
			 api {}
			 api {
			   endpoint "/**" {
			     response {}
			   }
			 }
			`,
			[]string{"/**"},
		},
		{
			"files + api: catch-all",
			`files {
			   base_path = "/public"
			   document_root = "."
			 }
			 api {
			   access_control = ["a"]
			 }
			`,
			[]string{"/**"},
		},
		{
			"files + api, same base path: no catch-all",
			`files {
			   document_root = "."
			 }
			 api {
			   access_control = ["a"]
			 }
			`,
			[]string{},
		},
		{
			"files + api, same base path: no catch-all",
			`files {
			   base_path = "/p"
			   document_root = "."
			 }
			 api {
			   base_path = "/p"
			   access_control = ["a"]
			   endpoint "/**" {
			     response {}
			   }
			 }
			`,
			[]string{"/p/**"},
		},
		{
			"spa + api: catch-all",
			`spa {
			   base_path = "/public"
			   bootstrap_file = "access_control_test.go"
			   paths = []
			 }
			 api {
			   access_control = ["a"]
			 }`,
			[]string{"/**"},
		},
		{
			"spa + api, same base path: no catch-all",
			`spa {
			    base_path = "/path"
				bootstrap_file = "access_control_test.go"
				paths = []
			 }
			 api {
			   base_path = "/path"
			   access_control = ["a"]
			 }
			`,
			[]string{},
		},
		{
			"files + api, same base path: no catch-all",
			`spa {
			   base_path = "/p"
			   bootstrap_file = "access_control_test.go"
			   paths = []
			 }
			 api {
			   base_path = "/p"
			   access_control = ["a"]
			   endpoint "/**" {
			     response {}
			   }
			 }
			`,
			[]string{"/p/**"},
		},
	}

	template := `
		server {
		  %%
		}
		definitions {
		  jwt "a" {
		    signature_algorithm = "HS256"
		    key = "asdf"
		  }
		}
	`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(strings.Replace(template, "%%", tt.hcl, -1)), "couper.hcl")
			if err != nil {
				subT.Error(err)
				return
			}
			log, _ := test.NewLogger()
			logger := log.WithContext(context.TODO())
			tmpStoreCh := make(chan struct{})
			defer close(tmpStoreCh)

			ctx, cancel := context.WithCancel(conf.Context)
			conf.Context = ctx
			defer cancel()

			server, err := runtime.NewServerConfiguration(conf, logger, cache.New(logger, tmpStoreCh))

			if err != nil {
				subT.Error("expected no error, got:", err)
				return
			}

			endpointMap := server[8080]["*"].EndpointRoutes
			var endpoints sort.StringSlice
			for endpoint := range endpointMap {
				endpoints = append(endpoints, endpoint)
			}
			endpoints.Sort()
			if fmt.Sprint(tt.endpoints) != fmt.Sprint(endpoints) {
				subT.Errorf("unexpected endpoints, want: %v, got: %v", tt.endpoints, endpoints)
				return
			}
		})
	}
}

// TestBasicAuthArgon2Warnings ensures the over-cap warning reaches the log once,
// with location, fact and advice. The -watch reload builds the configuration
// twice, and the dry run must stay silent.
func TestBasicAuthArgon2Warnings(t *testing.T) {
	const hcl = `
		server {}
		definitions {
		  basic_auth "ba" {
		    htpasswd_file = "../../accesscontrol/testdata/htpasswd_argon2_over_cap"
		  }
		}
	`

	const wantFirst = `basic_auth "ba": user "overm" (line 1): argon2 parameter m=94209 KiB exceeds the recommended maximum of 94208 KiB. Lower the parameter, or put a beta_rate_limiter before this access control.`

	for _, tt := range []struct {
		name     string
		dryRun   bool
		warnings int
	}{
		{"accepted configuration warns", false, 3},
		{"dry run stays silent", true, 0},
	} {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(hcl), "couper.hcl")
			if err != nil {
				subT.Fatal(err)
			}
			log, hook := test.NewLogger()
			logger := log.WithContext(context.TODO())
			tmpStoreCh := make(chan struct{})
			defer close(tmpStoreCh)

			// The -watch reload sets the dry run flag on the inner context, see main.go.
			ctx := context.Background()
			if tt.dryRun {
				ctx = context.WithValue(ctx, request.ConfigDryRun, true)
			}
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			conf.Context = conf.Context.Value(request.ContextType).(*eval.Context).WithContext(ctx)

			if _, err = runtime.NewServerConfiguration(conf, logger, cache.New(logger, tmpStoreCh)); err != nil {
				subT.Fatal(err)
			}

			var got []string
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.WarnLevel && strings.HasPrefix(entry.Message, `basic_auth "ba"`) {
					got = append(got, entry.Message)
				}
			}
			if len(got) != tt.warnings {
				subT.Fatalf("want %d warnings, got %d: %v", tt.warnings, len(got), got)
			}
			if len(got) > 0 && got[0] != wantFirst {
				subT.Errorf("want first warning %q, got %q", wantFirst, got[0])
			}
		})
	}
}

// TestBasicAuthArgon2Budget ensures one memory budget sizes the argon2 limiter
// for all basic_auth blocks, that Couper reports the result once, warns if one
// derivation alone exceeds the budget, and rejects an invalid budget.
func TestBasicAuthArgon2Budget(t *testing.T) {
	const template = `
		server {}
		settings {
		  %s
		}
		definitions {
		  basic_auth "a" {
		    htpasswd_file = "../../accesscontrol/testdata/htpasswd"
		  }
		  basic_auth "b" {
		    htpasswd_file = "../../accesscontrol/testdata/htpasswd_argon2_over_cap"
		  }
		}
	`
	// The most expensive entry has m=94209 KiB, so the default 256MiB hold two derivations.
	defaultSlots := min(2, goruntime.GOMAXPROCS(0))

	for _, tt := range []struct {
		name     string
		setting  string
		wantInfo string
		wantWarn string
		wantErr  string
	}{
		{"default budget", "", fmt.Sprintf("concurrent argon2 derivations: %d, peak memory: ", defaultSlots), "", ""},
		{"budget below one derivation", `beta_argon2_memory_budget = "64MiB"`, "concurrent argon2 derivations: 1,", "more than the beta_argon2_memory_budget of 64MiB", ""},
		{"invalid budget", `beta_argon2_memory_budget = "much"`, "", "", "beta_argon2_memory_budget"},
	} {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(fmt.Sprintf(template, tt.setting)), "couper.hcl")
			if err != nil {
				subT.Fatal(err)
			}
			log, hook := test.NewLogger()
			logger := log.WithContext(context.TODO())
			tmpStoreCh := make(chan struct{})
			defer close(tmpStoreCh)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conf.Context = conf.Context.Value(request.ContextType).(*eval.Context).WithContext(ctx)

			_, err = runtime.NewServerConfiguration(conf, logger, cache.New(logger, tmpStoreCh))
			if tt.wantErr != "" {
				var cErr *couperErr.Error
				if !goerrors.As(err, &cErr) || !strings.Contains(cErr.LogError(), tt.wantErr) {
					subT.Fatalf("want a configuration error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				subT.Fatal(err)
			}

			var infos, warns []string
			for _, entry := range hook.AllEntries() {
				if !strings.HasPrefix(entry.Message, "basic_auth: ") {
					continue
				}
				switch entry.Level {
				case logrus.InfoLevel:
					infos = append(infos, entry.Message)
				case logrus.WarnLevel:
					warns = append(warns, entry.Message)
				}
			}
			if len(infos) != 1 || !strings.Contains(infos[0], tt.wantInfo) {
				subT.Errorf("want one info containing %q, got: %v", tt.wantInfo, infos)
			}
			if tt.wantWarn == "" && len(warns) != 0 {
				subT.Errorf("want no budget warning, got: %v", warns)
			}
			if tt.wantWarn != "" && (len(warns) != 1 || !strings.Contains(warns[0], tt.wantWarn)) {
				subT.Errorf("want one warning containing %q, got: %v", tt.wantWarn, warns)
			}
		})
	}
}
