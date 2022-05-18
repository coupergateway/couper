package runtime_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/test"
)

func TestACDefinitions_errors(t *testing.T) {
	tests := []struct {
		name        string
		hcl         string
		expectedMsg string
	}{
		{
			"collision: basic_auth/jwt",
			`
			server "test" {
			}
			definitions {
				basic_auth "foo" {
				}
				jwt "foo" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					header = "Authorization"
				}
			}
			`,
			"configuration error: foo: accessControl already defined",
		},
		{
			"collision: jwt reserved label",
			`
			server "test" {
			}
			definitions {
				jwt "beta_granted_permissions" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					header = "Authorization"
				}
			}
			`,
			"configuration error: beta_granted_permissions: accessControl uses reserved name as label",
		},
		{
			"collision: basic_auth reserved label 1",
			`
			server "test" {
			}
			definitions {
				basic_auth "beta_granted_permissions" {
				}
			}
			`,
			"configuration error: beta_granted_permissions: accessControl uses reserved name as label",
		},
		{
			"collision: basic_auth reserved label 2",
			`
			server "test" {
			}
			definitions {
				basic_auth "beta_required_permission" {
				}
			}
			`,
			"configuration error: beta_required_permission: accessControl uses reserved name as label",
		},
		{
			"jwt with empty label",
			`
			server "test" {
			}
			definitions {
				jwt "" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					header = "Authorization"
				}
			}
			`,
			"configuration error: accessControl requires a label",
		},
		{
			"basic_auth with empty label",
			`
			server "test" {
			}
			definitions {
				basic_auth "" {
				}
			}
			`,
			"configuration error: accessControl requires a label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				subT.Fatal(err)
			}
			_, err = runtime.NewServerConfiguration(cf, nil, nil)
			if err == nil {
				subT.Errorf("Expected error")
			}
			logErr, _ := err.(errors.GoError)
			if logErr == nil {
				subT.Error("logErr should not be nil")
			} else if logErr.LogError() != tt.expectedMsg {
				subT.Errorf("\nwant:\t%s\ngot:\t%v", tt.expectedMsg, logErr.LogError())
			}
		})
	}
}

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
			   bootstrap_file = "foo"
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
				bootstrap_file = "foo"
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
			   bootstrap_file = "foo"
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
