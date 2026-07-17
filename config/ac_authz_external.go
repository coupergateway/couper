package config

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	hclbody "github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/config/meta"
)

var (
	_ BackendReference      = &AuthZExternal{}
	_ BackendInitialization = &AuthZExternal{}
	_ Body                  = &AuthZExternal{}
	_ Inline                = &AuthZExternal{}
)

// AuthZExternal represents the beta_authz_external block.
type AuthZExternal struct {
	ErrorHandlerSetter
	BackendName      string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the authorization callout. Mutually exclusive with {backend} block."`
	IncludeTLS       bool     `hcl:"include_tls,optional" docs:"Include TLS connection information of the client request in the authorization request." default:"false"`
	Name             string   `hcl:"name,label"`
	PermissionsClaim string   `hcl:"permissions_claim,optional" docs:"Name of the response body property containing the granted permissions. The property value must either be a string containing a space-separated list of permissions or a list of string permissions."`
	URL              string   `hcl:"url,optional" docs:"URL of the authorization service. Relative URL references are resolved against the origin of a referenced or nested {backend} block."`
	Remain           hcl.Body `hcl:",remain"`

	// Internally used
	Backend *hclsyntax.Body
}

func (a *AuthZExternal) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if err = a.check(); err != nil {
		return err
	}
	a.Backend, err = backendFunc("url", a.URL, a)
	return err
}

// check ensures a callout destination exists: a url or a backend providing an origin.
func (a *AuthZExternal) check() error {
	if a.URL == "" && a.BackendName == "" && len(hclbody.BlocksOfType(a.HCLBody(), "backend")) == 0 {
		return fmt.Errorf("url attribute or backend required")
	}
	return nil
}

// Reference implements the <BackendReference> interface.
func (a *AuthZExternal) Reference() string {
	return a.BackendName
}

// HCLBody implements the <Body> interface.
func (a *AuthZExternal) HCLBody() *hclsyntax.Body {
	return a.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (a *AuthZExternal) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend *Backend `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for the authorization callout (zero or one). Mutually exclusive with {backend} attribute."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (a *AuthZExternal) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(a)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(a.Inline())

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}
