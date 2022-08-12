package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ BackendReference = &TokenRequest{}
	_ Inline           = &TokenRequest{}
)

var TokenRequestBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:          "beta_token_request",
			LabelNames:    []string{"name"},
			LabelOptional: true,
		},
	},
}

type TokenRequest struct {
	BackendName string   `hcl:"backend,optional" docs:"backend block reference is required if no backend block is defined"`
	Name        string   `hcl:"name,label,optional"`
	URL         string   `hcl:"url,optional" docs:"If defined, the host part of the URL must be the same as the <code>origin</code> attribute of the <code>backend</code> block (if defined)."`
	Remain      hcl.Body `hcl:",remain"`

	// Internally used
	Backend hcl.Body
}

// Reference implements the <BackendReference> interface.
func (t *TokenRequest) Reference() string {
	return t.BackendName
}

// HCLBody implements the <Inline> interface.
func (t *TokenRequest) HCLBody() hcl.Body {
	return t.Remain
}

// Inline implements the <Inline> interface.
func (t *TokenRequest) Inline() interface{} {
	type Inline struct {
		Backend        *Backend             `hcl:"backend,block"`
		Body           string               `hcl:"body,optional" docs:"Creates implicit default <code>Content-Type: text/plain</code> header field"`
		ExpectedStatus []int                `hcl:"expected_status,optional" docs:"If defined, the response status code will be verified against this list of status codes, If the status code is unexpected a <code>beta_backend_token_request</code> error can be handled with an <code>error_handler</code>"`
		FormBody       string               `hcl:"form_body,optional" docs:"Creates implicit default <code>Content-Type: application/x-www-form-urlencoded</code> header field."`
		Headers        map[string]string    `hcl:"headers,optional" docs:"sets the given request headers"`
		JsonBody       string               `hcl:"json_body,optional" docs:"Creates implicit default <code>Content-Type: application/json</code> header field" type:"null, bool, number, string, object, tuple"`
		Method         string               `hcl:"method,optional" default:"GET"`
		QueryParams    map[string]cty.Value `hcl:"query_params,optional" docs:"sets the url query parameters"`
		TTL            string               `hcl:"ttl" docs:"The time span for which the token is to be stores."`
		Token          string               `hcl:"token" docs:"The token to be stored in <code>backends.<backend_name>.tokens.<token_request_name></code>"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (t *TokenRequest) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(t)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(t.Inline())

	// A backend reference is defined, backend block is not allowed.
	if t.BackendName != "" {
		schema.Blocks = nil
	}

	return schema
}
