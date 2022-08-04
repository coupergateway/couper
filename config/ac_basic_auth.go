package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/internal/seetie"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &BasicAuth{}

// BasicAuth represents the "basic_auth" config block
type BasicAuth struct {
	ErrorHandlerSetter
	File   string   `hcl:"htpasswd_file,optional" docs:"The htpasswd file."`
	Name   string   `hcl:"name,label"`
	User   string   `hcl:"user,optional" docs:"The user name."`
	Pass   string   `hcl:"password,optional" docs:"The corresponding password."`
	Realm  string   `hcl:"realm,optional" docs:"The realm to be sent in a WWW-Authenticate response HTTP header field."`
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface. Internally used for 'error_handler'.
func (b *BasicAuth) HCLBody() hcl.Body {
	return b.Remain
}

func (b *BasicAuth) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (b *BasicAuth) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(b)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(b.Inline())
	return schema
}

func (b *BasicAuth) DefaultErrorHandler() *ErrorHandler {
	wwwAuthenticateValue := "Basic"
	if b.Realm != "" {
		wwwAuthenticateValue += " realm=" + b.Realm
	}
	return &ErrorHandler{
		Kinds: []string{"basic_auth"},
		Remain: body.New(&hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"set_response_headers": {Name: "set_response_headers", Expr: hcl.StaticExpr(seetie.MapToValue(map[string]interface{}{
				"Www-Authenticate": wwwAuthenticateValue,
			}), hcl.Range{Filename: "default_basic_auth_error_handler"})}}}),
	}
}
