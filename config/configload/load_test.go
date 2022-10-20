package configload_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/test"
)

func TestPrepareBackendRefineAttributes(t *testing.T) {
	config := `server {
	endpoint "/" {
		request {
			backend "ref" {
				%s = env.VAR
			}
		}
	}
}

definitions {
	backend "ref" {
		origin = "http://localhost"
	}
}`

	for _, attribute := range []string{
		"disable_certificate_validation",
		"disable_connection_reuse",
		"http2",
		"max_connections",
	} {
		_, err := configload.LoadBytes([]byte(fmt.Sprintf(config, attribute)), "test.hcl")
		if err == nil {
			t.Fatal("expected an error")
		}

		if !strings.HasSuffix(err.Error(),
			fmt.Sprintf("backend reference: refinement for %q is not permitted; ", attribute)) {
			t.Error(err)
		}
	}
}

func TestPrepareBackendRefineBlocks(t *testing.T) {
	config := `server {
	endpoint "/" {
		request {
			backend "ref" {
				%s
			}
		}
	}
}

definitions {
	backend "ref" {
		origin = "http://localhost"
	}
}`

	tests := []struct {
		name string
		hcl  string
	}{
		{
			"openapi",
			`openapi { file = ""}`,
		},
		{
			"oauth2",
			`oauth2 {
  grant_type = "client_credentials"
  token_endpoint = ""
  client_id = "asdf"
  client_secret = "asdf"
}`,
		},
		{
			"beta_health",
			`beta_health {
}`,
		},
		{
			"beta_token_request",
			`beta_token_request {
  token = "asdf"
  ttl = "1s"
}`,
		},
		{
			"beta_token_request",
			`beta_token_request "name" {
  token = "asdf"
  ttl = "1s"
}`,
		},
		{
			"beta_rate_limit",
			`beta_rate_limit {
  per_period = 10
  period = "10s"
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := configload.LoadBytes([]byte(fmt.Sprintf(config, tt.hcl)), "test.hcl")
			if err == nil {
				subT.Error("expected an error")
				return
			}

			if !strings.HasSuffix(err.Error(),
				fmt.Sprintf("backend reference: refinement for %q is not permitted; ", tt.name)) {
				subT.Error(err)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"Bad interval",
			`interval = "10sec"`,
			`configuration error: foo: time: unknown unit "sec" in duration "10sec"`,
		},
		{
			"Bad timeout",
			`timeout = 1`,
			`configuration error: foo: time: missing unit in duration "1"`,
		},
		{
			"Bad threshold",
			`failure_threshold = -1`,
			"couper.hcl:13,29-30: Unsuitable value type; Unsuitable value: value must be a whole number, between 0 and 18446744073709551615 inclusive",
		},
		{
			"Bad expected status",
			`expected_status = 200`,
			"couper.hcl:13,27-30: Unsuitable value type; Unsuitable value: list of number required",
		},
		{
			"OK",
			`failure_threshold = 3
			 timeout = "3s"
			 interval = "5s"
			 expected_text = 123
			 expected_status = [200, 418]`,
			"",
		},
	}

	logger, _ := test.NewLogger()
	log := logger.WithContext(context.TODO())

	template := `
		server {
		  endpoint "/" {
		    proxy {
		      backend = "foo"
		    }
		  }
		}
		definitions {
		  backend "foo" {
		    origin = "..."
		    beta_health {
		      %%
		    }
		  }
		}`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(strings.Replace(template, "%%", tt.hcl, -1)), "couper.hcl")

			closeCh := make(chan struct{})
			defer close(closeCh)
			memStore := cache.New(log, closeCh)

			if conf != nil {
				ctx, cancel := context.WithCancel(conf.Context)
				conf.Context = ctx
				defer cancel()

				_, err = runtime.NewServerConfiguration(conf, log, memStore)
			}

			var errorMsg = ""
			if err != nil {
				if gErr, ok := err.(errors.GoError); ok {
					errorMsg = gErr.LogError()
				} else {
					errorMsg = err.Error()
				}
			}

			if tt.error != errorMsg {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, errorMsg)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"missing per_period",
			``,
			`Missing required argument; The argument "per_period" is required`,
		},
		{
			"missing period",
			`per_period = 10`,
			`Missing required argument; The argument "period" is required`,
		},
		{
			"OK",
			`period = "1m"
			 per_period = 10`,
			"",
		},
	}

	logger, _ := test.NewLogger()
	log := logger.WithContext(context.TODO())

	template := `
		server {}
		definitions {
		  backend "foo" {
		    beta_rate_limit {
		      %s
		    }
		  }
		}`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(fmt.Sprintf(template, tt.hcl)), "couper.hcl")

			closeCh := make(chan struct{})
			defer close(closeCh)
			memStore := cache.New(log, closeCh)

			if conf != nil {
				ctx, cancel := context.WithCancel(conf.Context)
				conf.Context = ctx
				defer cancel()

				_, err = runtime.NewServerConfiguration(conf, log, memStore)
			}

			var errorMsg = ""
			if err != nil {
				errorMsg = err.Error()
			}

			if !strings.Contains(errorMsg, tt.error) {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, errorMsg)
			}
		})
	}
}

func TestEndpointPaths(t *testing.T) {
	tests := []struct {
		name       string
		serverBase string
		apiBase    string
		endpoint   string
		expected   string
	}{
		{"only /", "", "", "/", "/"},
		{"simple path", "", "", "/pa/th", "/pa/th"},
		{"trailing /", "", "", "/pa/th/", "/pa/th/"},
		{"double /", "", "", "//", "//"},
		{"double /", "", "", "//path", "//path"},
		{"double /", "", "", "/pa//th", "/pa//th"},

		{"param", "", "", "/{param}", "/{param}"},

		{"server base_path /", "/", "", "/", "/"},
		{"server base_path /", "/", "", "/path", "/path"},
		{"server base_path", "/server", "", "/path", "/server/path"},
		{"server base_path with / endpoint", "/server", "", "/", "/server"},
		{"server base_path missing /", "server", "", "/path", "/server/path"},
		{"server base_path trailing /", "/server/", "", "/path", "/server/path"},
		{"server base_path double /", "/server", "", "//path", "/server//path"},
		{"server base_path trailing + double /", "/server/", "", "//path", "/server//path"},

		{"api base_path /", "", "/", "/", "/"},
		{"api base_path /", "", "/", "/path", "/path"},
		{"api base_path", "", "/api", "/path", "/api/path"},
		{"api base_path with / endpoint", "", "/api", "/", "/api"},
		{"api base_path missing /", "", "api", "/path", "/api/path"},
		{"api base_path trailing /", "", "/api/", "/path", "/api/path"},
		{"api base_path double /", "", "/api", "//path", "/api//path"},
		{"api base_path trailing + double /", "/api/", "", "//path", "/api//path"},

		{"server + api base_path /", "/", "/", "/", "/"},
		{"server + api base_path", "/server", "/api", "/", "/server/api"},
		{"server + api base_path", "/server", "/api", "/path", "/server/api/path"},
		{"server + api base_path missing /", "server", "api", "/", "/server/api"},
	}

	logger, _ := test.NewLogger()
	log := logger.WithContext(context.TODO())

	template := `
		server {
		  base_path = "%s"
		  api {
		    base_path = "%s"
		    endpoint "%s" {
		      response {}
		    }
		  }
		}`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			configBytes := []byte(fmt.Sprintf(template, tt.serverBase, tt.apiBase, tt.endpoint))
			config, err := configload.LoadBytes(configBytes, "couper.hcl")

			closeCh := make(chan struct{})
			defer close(closeCh)
			memStore := cache.New(log, closeCh)

			var serverConfig runtime.ServerConfiguration
			if err == nil {
				serverConfig, err = runtime.NewServerConfiguration(config, log, memStore)
			}

			if err != nil {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: <nil>\n\tGot:  %q", tt.name, err)
				return
			}

			var pattern string
			for key := range serverConfig[8080]["*"].EndpointRoutes {
				pattern = key
				break
			}

			if pattern != tt.expected {
				subT.Errorf("%q: Unexpected endpoint path:\n\tWant: %q\n\tGot:  %q", tt.name, tt.expected, pattern)
			}
		})
	}
}

func TestEnvironmentBlocksWithoutEnvironment(t *testing.T) {
	tests := []struct {
		name string
		hcl  string
		env  string
		want string
	}{
		{
			"no environment block, no setting",
			`
			definitions {}
			`,
			"",
			"configuration error: missing 'server' block",
		},

		{
			"environment block, but no setting",
			`
			environment "foo" {}
			server {}
			`,
			"",
			`"environment" blocks found, but "COUPER_ENVIRONMENT" setting is missing`,
		},
		{
			"environment block & setting",
			`
			environment "foo" {
			  server {}
			}
			`,
			"bar",
			"configuration error: missing 'server' block",
		},
		{
			"environment block & setting",
			`
			server {
			  environment "foo" {
			    endpoint "/" {}
			  }
			}
			`,
			"foo",
			"missing 'default' proxy or request block, or a response definition",
		},
		{
			"environment block & default setting",
			`
			environment "foo" {
			  server {}
			}
			settings {
				environment = "bar"
			}
			`,
			"",
			"configuration error: missing 'server' block",
		},
	}

	helper := test.New(t)

	file, err := os.CreateTemp("", "tmpfile-")
	helper.Must(err)
	defer file.Close()
	defer os.Remove(file.Name())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			config := []byte(tt.hcl)
			err := os.Truncate(file.Name(), 0)
			helper.Must(err)
			_, err = file.Seek(0, 0)
			helper.Must(err)
			_, err = file.Write(config)
			helper.Must(err)

			_, err = configload.LoadFile(file.Name(), tt.env)
			if err == nil && tt.want != "" {
				subT.Errorf("Missing expected error:\nWant:\t%q\nGot:\tnil", tt.want)
				return
			}

			if err != nil {
				message := err.Error()
				if !strings.Contains(message, tt.want) {
					subT.Errorf("Unexpected error message:\nWant:\t%q\nGot:\t%q", tt.want, message)
					return
				}
			}
		})
	}
}

func TestConfigErrors(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"websockets attribute and block",
			`proxy {
		      backend = "foo"
		      websockets = true
		      websockets {}
		    }`,
			"couper.hcl:6,9-26: either websockets attribute or block is allowed; ",
		},
	}

	template := `
		server {
		  endpoint "/" {
		    %%
		  }
		}`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := configload.LoadBytes([]byte(strings.Replace(template, "%%", tt.hcl, -1)), "couper.hcl")

			errorMsg := err.Error()
			if tt.error != errorMsg {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, errorMsg)
			}
		})
	}
}
