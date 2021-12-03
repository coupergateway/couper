package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ BackendReference = &Request{}
	_ Inline           = &Request{}
	_ SequenceItem     = &Request{}
)

type SequenceItem interface {
	Add(item SequenceItem)
	Deps() []SequenceItem
	GetName() string
}

// Request represents the <Request> object.
type Request struct {
	BackendName string   `hcl:"backend,optional"`
	Name        string   `hcl:"name,label"`
	Remain      hcl.Body `hcl:",remain"`

	// Internally used
	Backend hcl.Body
	depends []SequenceItem
}

// Requests represents a list of <Requests> objects.
type Requests []*Request

// Reference implements the <BackendReference> interface.
func (r Request) Reference() string {
	return r.BackendName
}

// HCLBody implements the <Inline> interface.
func (r Request) HCLBody() hcl.Body {
	return r.Remain
}

// Inline implements the <Inline> interface.
func (r Request) Inline() interface{} {
	type Inline struct {
		Backend     *Backend             `hcl:"backend,block"`
		Body        string               `hcl:"body,optional"`
		FormBody    string               `hcl:"form_body,optional"`
		JsonBody    string               `hcl:"json_body,optional"`
		Headers     map[string]string    `hcl:"headers,optional"`
		Method      string               `hcl:"method,optional"`
		QueryParams map[string]cty.Value `hcl:"query_params,optional"`
		URL         string               `hcl:"url,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (r Request) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(r.Inline())

	// A backend reference is defined, backend block is not allowed.
	if r.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, r.HCLBody())
}


func (r *Request) Add(item SequenceItem) {
	r.depends = append(r.depends, item)
}

func (r *Request) Deps() []SequenceItem {
	return r.depends[:]
}

func (r *Request) GetName() string {
	return r.Name
}
