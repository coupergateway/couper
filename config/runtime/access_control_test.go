package runtime_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/runtime"
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
