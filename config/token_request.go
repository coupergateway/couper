package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ BackendReference = &TokenRequest{}
	_ Body             = &TokenRequest{}
	_ Inline           = &TokenRequest{}
)

var tokenRequestBlockHeaderSchema = hcl.BlockHeaderSchema{
	Type:          "beta_token_request",
	LabelNames:    []string{"name"},
	LabelOptional: true,
}
var TokenRequestBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		tokenRequestBlockHeaderSchema,
	},
}

type TokenRequest struct {
	BackendName string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the token request. Mutually exclusive with {backend} block."`
	Name        string   `hcl:"name,label,optional"`
	URL         string   `hcl:"url,optional" docs:"URL of the resource to request the token from. May be relative to an origin specified in a referenced or nested {backend} block."`
	Remain      hcl.Body `hcl:",remain"`

	// Internally used
	Backend hcl.Body
}

// Reference implements the <BackendReference> interface.
func (t *TokenRequest) Reference() string {
	return t.BackendName
}

// HCLBody implements the <Body> interface.
func (t *TokenRequest) HCLBody() *hclsyntax.Body {
	return t.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (t *TokenRequest) Inline() interface{} {
	type Inline struct {
		Backend        *Backend             `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for the token request (zero or one). Mutually exclusive with {backend} attribute."`
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

	return schema
}
