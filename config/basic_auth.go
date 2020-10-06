package config

// BasicAuth represents the "basic_auth" config block
type BasicAuth struct {
	File  string `hcl:"htpasswd_file,optional"`
	Name  string `hcl:"name,label"`
	User  string `hcl:"user,optional"`
	Pass  string `hcl:"password,optional"`
	Realm string `hcl:"realm,optional"`
}
