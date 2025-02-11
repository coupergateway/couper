package config

import "github.com/hashicorp/hcl/v2"

type AuthZExternal struct {
	BackendName string   `hcl:"backend" docs:"References a default [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for authZ requests. Mutually exclusive with {backend} block."`
	IncludeTLS  bool     `hcl:"include_tls,optional"`
	Name        string   `hcl:"name,label"`
	Remain      hcl.Body `hcl:",remain"`
}

func (a *AuthZExternal) Prepare(backendFunc PrepareBackendFunc) error {
	//TODO implement me
	panic("implement me")
}
