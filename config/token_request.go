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

var TokenRequestBlockHeaderSchema = hcl.BlockHeaderSchema{
	Type:          "beta_token_request",
	LabelNames:    []string{"name"},
	LabelOptional: true,
}
var TokenRequestBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		TokenRequestBlockHeaderSchema,
	},
}

type TokenRequest struct {
	BackendName string   `hcl:"backend,optional" docs:"backend block reference is required if no backend block is defined"`
	Name        string   `hcl:"name,label,optional"`
	URL         string   `hcl:"url,optional" docs:"If defined, the host part of the URL must be the same as the {origin} attribute of the {backend} block (if defined)."`
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
		Body           string               `hcl:"body,optional" docs:"Creates implicit default {Content-Type: text/plain} header field"`
		ExpectedStatus []int                `hcl:"expected_status,optional" docs:"If defined, the response status code will be verified against this list of status codes, If the status code is unexpected a {beta_backend_token_request} error can be handled with an {error_handler}"`
		FormBody       string               `hcl:"form_body,optional" docs:"Creates implicit default {Content-Type: application/x-www-form-urlencoded} header field."`
		Headers        map[string]string    `hcl:"headers,optional" docs:"sets the given request headers"`
		JSONBody       string               `hcl:"json_body,optional" docs:"Creates implicit default {Content-Type: application/json} header field" type:"null, bool, number, string, object, tuple"`
		Method         string               `hcl:"method,optional" default:"GET"`
		QueryParams    map[string]cty.Value `hcl:"query_params,optional" docs:"sets the url query parameters"`
		TTL            string               `hcl:"ttl" docs:"The time span for which the token is to be stored."`
		Token          string               `hcl:"token" docs:"The token to be stored in {backends.<backend_name>.tokens.<token_request_name>}"`
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

	// TODO: check backend attribute vs backend block conflict at configload

	return schema
}
