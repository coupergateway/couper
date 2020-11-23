package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type Backend struct {
	ConnectTimeout   string   `hcl:"connect_timeout,optional"`
	Name             string   `hcl:"name,label"`
	Options          hcl.Body `hcl:",remain"`
	RequestBodyLimit string   `hcl:"request_body_limit,optional"`
	TTFBTimeout      string   `hcl:"ttfb_timeout,optional"`
	Timeout          string   `hcl:"timeout,optional"`
}

func (b Backend) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(b)
	if !inline {
		return schema
	}

	type Inline struct {
		Origin          string            `hcl:"origin,optional"`
		Hostname        string            `hcl:"hostname,optional"`
		Path            string            `hcl:"path,optional"`
		RequestHeaders  map[string]string `hcl:"request_headers,optional"`
		ResponseHeaders map[string]string `hcl:"response_headers,optional"`
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

	if result.Options != nil {
		bodies = append(bodies, result.Options)
	}

	if other.Options != nil {
		bodies = append(bodies, other.Options)
		result.Options = other.Options
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
