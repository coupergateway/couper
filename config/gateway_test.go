package config_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"go.avenga.cloud/couper/gateway/config"
)

func TestWriteGateway(t *testing.T) {
	conf := config.Gateway{
		Frontends: []*config.Frontend{
			{
				BasePath: "/hans/v1",
				Name:     "wurst",
				Endpoint: []*config.Endpoint{
					{
						Backend: &config.Backend{Type: "proxy"},
						Path:    "/proxy/"},
				},
			},
		},
	}
	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(&conf, f.Body())
	fmt.Printf("%s", f.Bytes())
}
