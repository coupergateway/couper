package config

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/internal/seetie"
)

// Internally used for 'error_handler'.
var _ Body = &BasicAuth{}

// BasicAuth represents the "basic_auth" config block
type BasicAuth struct {
	ErrorHandlerSetter
	File  string `hcl:"htpasswd_file,optional"`
	Name  string `hcl:"name,label"`
	User  string `hcl:"user,optional"`
	Pass  string `hcl:"password,optional"`
	Realm string `hcl:"realm,optional"`

	// Internally used for 'error_handler'.
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface. Internally used for 'error_handler'.
func (b *BasicAuth) HCLBody() hcl.Body {
	return b.Remain
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
