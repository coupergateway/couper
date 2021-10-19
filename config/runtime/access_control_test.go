package runtime

import (
	"testing"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/errors"
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
				jwt "scopes" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					header = "Authorization"
				}
			}
			`,
			"configuration error: scopes: accessControl uses reserved name as label",
		},
		{
			"collision: basic_auth reserved label",
			`
			server "test" {
			}
			definitions {
				basic_auth "scopes" {
				}
			}
			`,
			"configuration error: scopes: accessControl uses reserved name as label",
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
			_, err = NewServerConfiguration(cf, nil, nil)
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
