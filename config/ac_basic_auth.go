package config

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/internal/seetie"
)

// BasicAuth represents the "basic_auth" config block
type BasicAuth struct {
	AccessControlSetter
	File   string   `hcl:"htpasswd_file,optional"`
	Name   string   `hcl:"name,label"`
	User   string   `hcl:"user,optional"`
	Pass   string   `hcl:"password,optional"`
	Realm  string   `hcl:"realm,optional"`
	Remain hcl.Body `hcl:",remain"`
}

func (b *BasicAuth) HCLBody() hcl.Body {
	return b.Remain
}

func (b *BasicAuth) DefaultErrorHandler() ([]string, hcl.Body) {
	wwwAuthenticateValue := "Basic"
	if b.Realm != "" {
		wwwAuthenticateValue += " realm=" + b.Realm
	}
	return []string{"basic_auth"},
		body.New(&hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"set_response_headers": {Name: "set_response_headers", Expr: hcl.StaticExpr(seetie.MapToValue(map[string]interface{}{
				"Www-Authenticate": wwwAuthenticateValue,
			}), hcl.Range{Filename: "default_basic_auth_error_handler"})}}})
}
