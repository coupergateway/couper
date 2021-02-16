package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Response{}

// Response represents the <Response> object.
type Response struct {
	Body   string   `hcl:"body,optional"`
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (r Response) HCLBody() hcl.Body {
	return r.Remain
}

// Reference implements the <Inline> interface.
func (r Response) Reference() string {
	return r.Name
}

// Schema implements the <Inline> interface.
func (r Response) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	type Inline struct {
		Status     int               `hcl:"status"`
		SetHeaders map[string]string `hcl:"set_headers,optional"`
		AddHeaders map[string]string `hcl:"add_headers,optional"`
		DelHeaders []string          `hcl:"remove_headers,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}

func newResponseSchema(schema *hcl.BodySchema, body hcl.Body) *hcl.BodySchema {
	for i, block := range schema.Blocks {
		// Inline response block MAY have no label.
		if block.Type == "response" && len(block.LabelNames) > 0 {
			// Check if a response block could be parsed w/ label, otherwise its an inline one w/o label.
			content, _, _ := body.PartialContent(schema)
			if content == nil || len(content.Blocks) == 0 {
				schema.Blocks[i].LabelNames = nil

				break
			}
		}
	}

	return schema
}
