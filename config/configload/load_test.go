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
