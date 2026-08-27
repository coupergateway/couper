package runtime_test

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/runtime"
	"github.com/coupergateway/couper/internal/test"
)

func TestAuthZenAnonymousSubjectWarning(t *testing.T) {
	tests := []struct {
		name    string
		hcl     string
		warning string
	}{
		{
			"basic_auth before beta_authzen without subject",
			`server {
			   api {
			     access_control = ["ba", "authz"]
			     endpoint "/" {
			       response {}
			     }
			   }
			 }`,
			`beta_authzen "authz" runs behind basic_auth "ba" (request.context.ba.user) without a subject attribute`,
		},
		{
			"chain split over server and endpoint",
			`server {
			   access_control = ["ba"]
			   endpoint "/" {
			     access_control = ["authz"]
			     response {}
			   }
			 }`,
			`beta_authzen "authz" runs behind basic_auth "ba" (request.context.ba.user) without a subject attribute`,
		},
		{
			"subject attribute set",
			`server {
			   api {
			     access_control = ["ba", "authz_subject"]
			     endpoint "/" {
			       response {}
			     }
			   }
			 }`,
			"",
		},
		{
			"jwt before beta_authzen",
			`server {
			   api {
			     access_control = ["token", "authz"]
			     endpoint "/" {
			       response {}
			     }
			   }
			 }`,
			"",
		},
		{
			"beta_authzen before basic_auth",
			`server {
			   api {
			     access_control = ["authz", "ba"]
			     endpoint "/" {
			       response {}
			     }
			   }
			 }`,
			"",
		},
		{
			"basic_auth disabled at the endpoint",
			`server {
			   access_control = ["ba"]
			   endpoint "/" {
			     disable_access_control = ["ba"]
			     access_control = ["authz"]
			     response {}
			   }
			 }`,
			"",
		},
	}

	template := `
		%%
		definitions {
		  basic_auth "ba" {
		    password = "asdf"
		  }
		  jwt "token" {
		    signature_algorithm = "HS256"
		    key = "asdf"
		  }
		  beta_authzen "authz" {
		    url = "http://localhost:8081/access/v1/evaluation"
		  }
		  beta_authzen "authz_subject" {
		    url = "http://localhost:8081/access/v1/evaluation"
		    subject = {
		      type = "identity"
		      id   = request.context.ba.user
		    }
		  }
		}
	`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(strings.Replace(template, "%%", tt.hcl, 1)), "couper.hcl")
			if err != nil {
				subT.Fatal(err)
			}
			log, hook := test.NewLogger()
			logger := log.WithContext(context.TODO())
			tmpStoreCh := make(chan struct{})
			defer close(tmpStoreCh)

			ctx, cancel := context.WithCancel(conf.Context)
			conf.Context = ctx
			defer cancel()

			if _, err = runtime.NewServerConfiguration(conf, logger, cache.New(logger, tmpStoreCh)); err != nil {
				subT.Fatal(err)
			}

			var warnings []string
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.WarnLevel {
					warnings = append(warnings, entry.Message)
				}
			}

			if tt.warning == "" {
				if len(warnings) > 0 {
					subT.Errorf("expected no warning, got: %v", warnings)
				}
				return
			}
			if len(warnings) != 1 {
				subT.Fatalf("expected exactly one warning, got: %v", warnings)
			}
			if !strings.Contains(warnings[0], tt.warning) {
				subT.Errorf("unexpected warning,\nwant: %s\ngot:  %s", tt.warning, warnings[0])
			}
		})
	}
}
