package configload_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/avenga/couper/config/configload"
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

	_, err := configload.LoadBytes([]byte(fmt.Sprintf(config, `openapi { file = ""}`)), "test.hcl")
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.HasSuffix(err.Error(),
		fmt.Sprintf("backend reference: refinement for %q is not permitted; ", "openapi")) {
		t.Error(err)
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
			`time: unknown unit "sec" in duration "10sec"`,
		},
		{
			"Bad timeout",
			`timeout = 1`,
			`time: missing unit in duration "1"`,
		},
		{
			"Bad failure_threshold",
			`failure_threshold = -1`,
			"couper.hcl:13,29-30: Unsuitable value type; Unsuitable value: value must be a whole number, between 0 and 18446744073709551615 inclusive",
		},
		{
			"Bad expect status",
			`expect_status = [200, 204]`,
			"couper.hcl:13,25-26: Unsuitable value type; Unsuitable value: number required",
		},
		{
			"OK",
			`failure_threshold = 3
			 timeout = "3s"
			 interval = "5s"
			 expect_text = 123
			 expect_status = 200`,
			"",
		},
	}

	logger, _ := logrustest.NewNullLogger()
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
			conf, err := LoadBytes([]byte(strings.Replace(template, "%%", tt.hcl, -1)), "couper.hcl")
			if conf != nil {
				_, err = runtime.NewServerConfiguration(conf, log, nil)
			}

			var error = ""
			if err != nil {
				error = err.Error()
			}

			if tt.error != error {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, error)
			}
		})
	}
}
