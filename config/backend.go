package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &Backend{}

type Backend struct {
	ConnectTimeout   string   `hcl:"connect_timeout,optional"`
	Name             string   `hcl:"name,label"`
	Remain           hcl.Body `hcl:",remain"`
	RequestBodyLimit string   `hcl:"request_body_limit,optional"`
	TTFBTimeout      string   `hcl:"ttfb_timeout,optional"`
	Timeout          string   `hcl:"timeout,optional"`
}

func (b Backend) Body() hcl.Body {
	return b.Remain
}

func (b Backend) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(b)
	if !inline {
		return schema
	}

	type Inline struct {
		Origin             string               `hcl:"origin,optional"`
		Hostname           string               `hcl:"hostname,optional"`
		Path               string               `hcl:"path,optional"`
		RequestHeaders     map[string]string    `hcl:"request_headers,optional"`
		ResponseHeaders    map[string]string    `hcl:"response_headers,optional"`
		SetRequestHeaders  map[string]string    `hcl:"set_request_headers,optional"`
		SetResponseHeaders map[string]string    `hcl:"set_response_headers,optional"`
		AddQueryParams     map[string]cty.Value `hcl:"add_query_params,optional"`
		DelQueryParams     []string             `hcl:"remove_query_params,optional"`
		SetQueryParams     map[string]cty.Value `hcl:"set_query_params,optional"`
	}

	schema, _ = gohcl.ImpliedBodySchema(&Inline{})
	return schema
}

// Merge overrides the left backend configuration and returns a new instance.
func (b *Backend) Merge(other *Backend) (*Backend, []hcl.Body) {
	if b == nil || other == nil {
		return nil, nil
	}

	var bodies []hcl.Body

	result := *b

	if other.Name != "" {
		result.Name = other.Name
	}

	if result.Remain != nil {
		bodies = append(bodies, result.Remain)
	}

	if other.Remain != nil {
		bodies = append(bodies, other.Remain)
		result.Remain = other.Remain
	}

	if other.ConnectTimeout != "" {
		result.ConnectTimeout = other.ConnectTimeout
	}

	if other.RequestBodyLimit != "" {
		result.RequestBodyLimit = other.RequestBodyLimit
	}

	if other.TTFBTimeout != "" {
		result.TTFBTimeout = other.TTFBTimeout
	}

	if other.Timeout != "" {
		result.Timeout = other.Timeout
	}

	return &result, bodies
}

func newBackendSchema(schema *hcl.BodySchema, body hcl.Body) *hcl.BodySchema {
	for i, block := range schema.Blocks {
		// inline backend block MAY have no label
		if block.Type == "backend" && len(block.LabelNames) > 0 {
			// check if a backend block could be parsed with label, otherwise its an inline one without label.
			content, _, _ := body.PartialContent(schema)
			if content == nil || len(content.Blocks) == 0 {
				schema.Blocks[i].LabelNames = nil
			}
		}
	}
	return schema
}
