package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	hclbody "github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/config/meta"
)

var (
	_ BackendReference      = &ExternalAuthZ{}
	_ BackendInitialization = &ExternalAuthZ{}
	_ Body                  = &ExternalAuthZ{}
	_ Inline                = &ExternalAuthZ{}
)

// ExternalAuthZ represents the beta_external_authz block.
type ExternalAuthZ struct {
	ErrorHandlerSetter
	BackendName           string   `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the authorization callout. Mutually exclusive with {backend} block."`
	ConfigurationMaxStale string   `hcl:"configuration_max_stale,optional" docs:"Time after the expiration of the AuthZEN configuration document during which Couper keeps using it. A zero value means no stale use." type:"duration" default:"1h"`
	ConfigurationTTL      string   `hcl:"configuration_ttl,optional" docs:"Time to cache the AuthZEN configuration document." type:"duration" default:"1h"`
	ConfigurationURL      string   `hcl:"configuration_url,optional" docs:"URL of the AuthZEN configuration document ({/.well-known/authzen-configuration}) of the authorization service. Couper reads the callout endpoint from it. Mutually exclusive with {url}."`
	EvaluatePermissions   []string `hcl:"evaluate_permissions,optional" docs:"Candidate permissions to resolve with one batch callout to the AuthZEN access evaluations endpoint. Couper asks the authorization service about the client request and about every listed permission, and grants those it allows. A {required_permission} of the protected endpoint or API replaces the candidates for that request. Mutually exclusive with {permissions_property}."`
	IncludeTLS            bool     `hcl:"include_tls,optional" docs:"Include TLS connection information of the client request in the authorization request." default:"false"`
	Name                  string   `hcl:"name,label"`
	PermissionsProperty   string   `hcl:"permissions_property,optional" docs:"Name of the property in the response {context} containing the granted permissions. The property value must either be a string containing a space-separated list of permissions or a list of string permissions."`
	URL                   string   `hcl:"url,optional" docs:"URL of the authorization service. Relative URL references are resolved against the origin of a referenced or nested {backend} block. Without a path, or with only the root path {/}, the AuthZEN access evaluation endpoint {/access/v1/evaluation} is used — or {/access/v1/evaluations} with {evaluate_permissions}. An explicit path must point to the matching endpoint." default:"/access/v1/evaluation"`
	Remain                hcl.Body `hcl:",remain"`

	// Internally used
	Backend *hclsyntax.Body
}

func (a *ExternalAuthZ) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if err = a.check(); err != nil {
		return err
	}
	// Discovery and the callout share one backend, because the discovered endpoint has to
	// stay on the origin of the configuration document anyway.
	if a.ConfigurationURL != "" {
		a.Backend, err = backendFunc("configuration_url", a.ConfigurationURL, a)
		return err
	}
	a.Backend, err = backendFunc("url", a.URL, a)
	return err
}

// check ensures a callout destination exists: a url or a backend providing an origin.
func (a *ExternalAuthZ) check() error {
	if a.URL != "" && a.ConfigurationURL != "" {
		return fmt.Errorf("url and configuration_url are mutually exclusive")
	}
	if a.URL == "" && a.ConfigurationURL == "" && a.BackendName == "" &&
		len(hclbody.BlocksOfType(a.HCLBody(), "backend")) == 0 {
		return fmt.Errorf("url attribute or backend required")
	}
	for _, permission := range a.EvaluatePermissions {
		if strings.TrimSpace(permission) == "" {
			return fmt.Errorf("evaluate_permissions must not contain empty entries")
		}
	}
	// Both fill the granted permissions, but from different callouts; combining them would
	// need a second round trip on the hot path.
	if a.PermissionsProperty != "" && len(a.EvaluatePermissions) > 0 {
		return fmt.Errorf("permissions_property and evaluate_permissions are mutually exclusive")
	}
	return nil
}

// Reference implements the <BackendReference> interface.
func (a *ExternalAuthZ) Reference() string {
	return a.BackendName
}

// HCLBody implements the <Body> interface.
func (a *ExternalAuthZ) HCLBody() *hclsyntax.Body {
	return a.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface. The AuthZEN entities are evaluated per request,
// so a preceding access control can name the subject.
func (a *ExternalAuthZ) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Action   map[string]cty.Value `hcl:"action,optional" docs:"Replaces the action of the access evaluation request. Requires a {name}; an optional {properties} object is passed through. Defaults to the request method."`
		Backend  *Backend             `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for the authorization callout (zero or one). Mutually exclusive with {backend} attribute."`
		Context  map[string]cty.Value `hcl:"context,optional" docs:"Merges into the context of the access evaluation request. Configured keys win over the {headers} and {tls} defaults."`
		Resource map[string]cty.Value `hcl:"resource,optional" docs:"Replaces the resource of the access evaluation request. Requires a {type} and an {id}; an optional {properties} object is passed through. Defaults to the matched route."`
		Subject  map[string]cty.Value `hcl:"subject,optional" docs:"Replaces the subject of the access evaluation request. Requires a {type} and an {id}; an optional {properties} object is passed through. Defaults to the bearer token of the client request."`
	}

	return &Inline{}
}

// DefaultErrorHandlers forwards the authorization service's WWW-Authenticate challenge
// on denied credentials so clients can bootstrap authentication (e.g. OAuth protected
// resource metadata discovery); a user-defined handler for the kind replaces it.
func (a *ExternalAuthZ) DefaultErrorHandlers() []*ErrorHandler {
	challenge := &hclsyntax.ScopeTraversalExpr{
		Traversal: hcl.Traversal{
			hcl.TraverseRoot{Name: "request"},
			hcl.TraverseAttr{Name: "context"},
			hcl.TraverseAttr{Name: a.Name},
			hcl.TraverseAttr{Name: "www_authenticate"},
		},
	}
	headers := &hclsyntax.ObjectConsExpr{
		Items: []hclsyntax.ObjectConsItem{
			{
				KeyExpr: &hclsyntax.ObjectConsKeyExpr{
					Wrapped: &hclsyntax.LiteralValueExpr{Val: cty.StringVal("Www-Authenticate")},
				},
				ValueExpr: challenge,
			},
		},
	}
	return []*ErrorHandler{
		{
			Kinds: []string{"external_authz_invalid_credentials"},
			Remain: &hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"set_response_headers": {
						Name:     "set_response_headers",
						Expr:     headers,
						SrcRange: hcl.Range{Filename: "default_external_authz_error_handler"},
					},
				},
			},
		},
	}
}

// Schema implements the <Inline> interface.
func (a *ExternalAuthZ) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(a)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(a.Inline())

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}
