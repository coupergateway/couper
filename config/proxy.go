package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference = &Proxy{}
	_ Body             = &Proxy{}
	_ Inline           = &Proxy{}
)

// Proxy represents the <Proxy> object.
type Proxy struct {
	BackendName string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the proxy request. Mutually exclusive with {backend} block."`
	Name        string   `hcl:"name,label,optional"`
	Remain      hcl.Body `hcl:",remain"`
	ReqName     string   `hcl:"name,optional" docs:"Defines the proxy request name. Allowed only in the [{definitions} block](definitions)." default:"default"`
	Websockets  *bool    `hcl:"websockets,optional" docs:"Allows support for WebSockets. This attribute is only allowed in the \"default\" proxy block. Other {proxy} blocks, {request} blocks or {response} blocks are not allowed within the current {endpoint} block. Mutually exclusive with {websockets} block."`

	// internally used
	Backend *hclsyntax.Body
}

// Proxies represents a list of <Proxy> objects.
type Proxies []*Proxy

// Reference implements the <BackendReference> interface.
func (p Proxy) Reference() string {
	return p.BackendName
}

// HCLBody implements the <Body> interface.
func (p Proxy) HCLBody() *hclsyntax.Body {
	return p.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (p Proxy) Inline() interface{} {
	type Inline struct {
		meta.RequestHeadersAttributes
		meta.ResponseHeadersAttributes
		meta.FormParamsAttributes
		meta.QueryParamsAttributes
		Backend        *Backend    `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for the proxy request (zero or one). Mutually exclusive with {backend} attribute."`
		ExpectedStatus []int       `hcl:"expected_status,optional" docs:"If defined, the response status code will be verified against this list of codes. If the status code not included in this list an {unexpected_status} error will be thrown which can be handled with an [{error_handler}](error_handler)."`
		URL            string      `hcl:"url,optional" docs:"URL of the resource to request. May be relative to an origin specified in a referenced or nested {backend} block."`
		Websockets     *Websockets `hcl:"websockets,block" docs:"Configures support for [websockets](/configuration/block/websockets) connections (zero or one). Mutually exclusive with {websockets} attribute."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (p Proxy) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(p)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(p.Inline())

	return meta.MergeSchemas(schema, meta.ModifierAttributesSchema)
}
