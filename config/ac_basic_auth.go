package config

import "github.com/hashicorp/hcl/v2"

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
