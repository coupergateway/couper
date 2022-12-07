package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var (
	_ BackendInitialization = &Introspection{}
	_ Body                  = &Introspection{}
	_ Inline                = &Introspection{}
)

type Introspection struct {
	BackendName string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for introspection requests. Mutually exclusive with {backend} block."`
	Endpoint    string   `hcl:"endpoint" docs:"The authorization server's {introspection_endpoint}."`
	Interval    string   `hcl:"interval" docs:"The interval after which a new introspection request for a token is sent (= TTL of cached introspection responses)."`
	Remain      hcl.Body `hcl:",remain"`

	// Internally used
	Backend         *hclsyntax.Body
	IntervalSeconds int64
}

// TODO use function from config/configload
func newDiagErr(subject *hcl.Range, summary string) error {
	return hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Subject:  subject,
	}}
}

func (i *Introspection) Prepare(backendFunc PrepareBackendFunc) error {
	b, err := backendFunc("introspection_endpoint", i.Endpoint, i)
	if err != nil {
		return err
	}

	i.Backend = b

	attrs := i.Remain.(*hclsyntax.Body).Attributes
	r := attrs["interval"].Expr.Range()

	dur, err := ParseDuration("interval", i.Interval, -1)
	if err != nil {
		return newDiagErr(&r, err.Error())
	} else if dur == -1 {
		return newDiagErr(&r, "invalid duration")
	}

	i.IntervalSeconds = int64(dur.Seconds())

	return nil
}

// Reference implements the <BackendReference> interface.
func (i *Introspection) Reference() string {
	return i.BackendName
}

// HCLBody implements the <Body> interface.
func (i *Introspection) HCLBody() *hclsyntax.Body {
	return i.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (i *Introspection) Inline() interface{} {
	type Inline struct {
		// meta.LogFieldsAttribute
		Backend *Backend `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for introspection requests (zero or one). Mutually exclusive with {backend} attribute."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (i *Introspection) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(i)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(i.Inline())

	return schema
	// return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}
