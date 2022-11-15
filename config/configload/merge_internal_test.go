package configload

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/internal/test"
)

func Test_mergeServers_ServerTLS(t *testing.T) {
	tests := []struct {
		name    string
		content []string
		want    string
		wantErr bool
	}{
		{"normal, one file", []string{`server {
  tls {
    server_certificate {
      attr1 = "val1"
    }
  }
}`}, `server {
  tls {
    server_certificate {
      attr1 = "val1"
    }
  }
}
`, false},
		{"two files, override", []string{`server {
  tls {
    server_certificate {
      attr1 = "val1"
    }
  }
}`, `server {
  tls {
    server_certificate {
      attr2 = "val2"
    }
  }
}`}, `server {
  tls {
    server_certificate {
      attr2 = "val2"
    }
  }
}
`, false},
		{"two files, merge", []string{`server {
  tls {
    server_certificate "example1.com" {
      attr1 = "val1"
    }
  }
}`, `server {
  tls {
    server_certificate "example2.com" {
      attr2 = "val2"
    }
  }
}`}, `server {
  tls {
    server_certificate "example1.com" {
      attr1 = "val1"
    }
    server_certificate "example2.com" {
      attr2 = "val2"
    }
  }
}
`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			hlp := test.New(st)

			parsedBodies, err := parseBodies(tt.content)
			hlp.Must(err)

			blocks, err := mergeServers(parsedBodies, nil)
			hlp.Must(err)

			result := writeBlocks(blocks)

			if (err != nil) != tt.wantErr {
				t.Errorf("bodiesToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(result, tt.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func Test_mergeDefinitions_BackendTLS(t *testing.T) {
	tests := []struct {
		name    string
		content []string
		want    string
		wantErr bool
	}{
		{"normal, one file", []string{`definitions {
  backend "one" {
    tls {
      server_ca_certificate_file = "same.crt"
    }
  }
}`}, `definitions {
  backend "one" {
    tls {
      server_ca_certificate_file = "same.crt"
    }
  }
}
`, false},
		{"two files, replace", []string{`definitions {
  backend "one" {
    origin = "https://localhost"
    tls {
      server_ca_certificate_file = "one.crt"
    }
  }
}`, `definitions {
  backend "one" {
    tls {
      server_ca_certificate_file = "two.crt"
    }
  }
}`}, `definitions {
  backend "one" {
    tls {
      server_ca_certificate_file = "two.crt"
    }
  }
}
`, false},
		{"two files, append", []string{`definitions {
  backend "one" {
    tls {
      server_ca_certificate_file = "one.crt"
    }
  }
}`, `definitions {
  backend "two" {
    tls {
      server_ca_certificate_file = "two.crt"
    }
  }
}`}, `definitions {
  backend "one" {
    tls {
      server_ca_certificate_file = "one.crt"
    }
  }
  backend "two" {
    tls {
      server_ca_certificate_file = "two.crt"
    }
  }
}
`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			hlp := test.New(st)

			parsedBodies, err := parseBodies(tt.content)
			hlp.Must(err)

			block, _, err := mergeDefinitions(parsedBodies)
			hlp.Must(err)

			result := writeBlocks(hclsyntax.Blocks{block})

			if (err != nil) != tt.wantErr {
				t.Errorf("bodiesToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(result, tt.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func writeBlocks(blocks hclsyntax.Blocks) string {
	f := hclwrite.NewEmptyFile()
	root := f.Body()

	for _, block := range blocks {
		appendBlock(root, block)
	}

	b := &bytes.Buffer{}
	_, _ = f.WriteTo(b)
	return b.String()
}

func parseBodies(bodies []string) ([]*hclsyntax.Body, error) {
	var parsedBodies []*hclsyntax.Body
	for _, bodyStr := range bodies {
		body, err := parser.Load([]byte(bodyStr), "")
		if err != nil {
			return nil, err
		}
		parsedBodies = append(parsedBodies, body)
	}
	return parsedBodies, nil
}

func appendBlock(parent *hclwrite.Body, block *hclsyntax.Block) {
	writeBlock := gohcl.EncodeAsBlock(block, block.Type)
	writeBlock.SetLabels(block.Labels)

	for _, subBlock := range block.Body.Blocks {
		appendBlock(writeBlock.Body(), subBlock)
	}

	appendAttrs(writeBlock.Body(), block.Body.Attributes)

	parent.AppendBlock(writeBlock)
}

func appendAttrs(parent *hclwrite.Body, attributes hclsyntax.Attributes) {
	for _, attr := range attributes {
		v, _ := attr.Expr.Value(&hcl.EvalContext{})
		parent.SetAttributeValue(attr.Name, v)
	}
}
