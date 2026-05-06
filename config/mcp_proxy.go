package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/meta"
)

var (
	_ BackendReference = &MCPProxy{}
	_ Body             = &MCPProxy{}
	_ Inline           = &MCPProxy{}
)

// MCPProxy represents the <MCPProxy> object.
type MCPProxy struct {
	BackendName string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the MCP proxy request. Mutually exclusive with {backend} block."`
	Name        string   `hcl:"name,label_optional"`
	Remain      hcl.Body `hcl:",remain"`

	// internally used
	Backend *hclsyntax.Body
}

// MCPProxies represents a list of <MCPProxy> objects.
type MCPProxies []*MCPProxy

// Reference implements the <BackendReference> interface.
func (m MCPProxy) Reference() string {
	return m.BackendName
}

// HCLBody implements the <Body> interface.
func (m MCPProxy) HCLBody() *hclsyntax.Body {
	return m.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (m MCPProxy) Inline() interface{} {
	type Inline struct {
		meta.RequestHeadersAttributes
		meta.ResponseHeadersAttributes
		meta.FormParamsAttributes
		meta.QueryParamsAttributes
		Backend      *Backend `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for the MCP proxy request (zero or one). Mutually exclusive with {backend} attribute."`
		AllowedTools []string `hcl:"allowed_tools,optional" docs:"List of tool name patterns (glob) to allow. If set, only matching tools are exposed."`
		BlockedTools []string `hcl:"blocked_tools,optional" docs:"List of tool name patterns (glob) to block. Matching tools are hidden."`
		URL          string   `hcl:"url,optional" docs:"URL of the resource to request. May be relative to an origin specified in a referenced or nested {backend} block."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (m MCPProxy) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(m)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(m.Inline())

	return meta.MergeSchemas(schema, meta.ModifierAttributesSchema)
}
