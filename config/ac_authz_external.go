package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type OpenFGAEntity struct {
	Namespace string         `hcl:"namespace" docs:"The namespace of the entity."`
	Name      hcl.Expression `hcl:"name" docs:"The name or identifier of the entity."`
}

type OpenFGAEntityRelation struct {
	Name hcl.Expression `hcl:"name" docs:"The name of the relation."`
}

type AuthZOpenFGA struct {
	User     *OpenFGAEntity         `hcl:"user,block" docs:"The user entity."`
	Relation *OpenFGAEntityRelation `hcl:"relation,block" docs:"The relation name."`
	Object   *OpenFGAEntity         `hcl:"object,block" docs:"The object entity."`
	StoreID  string                 `hcl:"store_id" docs:"The store ID to use for authorization against the OpenFGA server."`
	ModelID  string                 `hcl:"model_id,optional" docs:"The model ID to use for authorization against the OpenFGA store. If omitted, the latest store model is used."`
	Remain   hcl.Body               `hcl:",remain"`
}

type AuthZExternal struct {
	BackendName string        `hcl:"backend" docs:"References a default [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for authZ requests. Mutually exclusive with {backend} block."`
	URL         string        `hcl:"url,optional" docs:"The URL to call for authorization."`
	IncludeTLS  bool          `hcl:"include_tls,optional" docs:"Include TLS information in the authorization request."`
	Name        string        `hcl:"name,label" docs:"The name of the authorization."`
	OpenFGA     *AuthZOpenFGA `hcl:"openfga,block" docs:"Configure an [OpenFGA](/configuration/block/authz_external/openfga) authorization."`
	Remain      hcl.Body      `hcl:",remain"`

	// Internally used
	Backend *hclsyntax.Body
}

func (a *AuthZExternal) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if a.URL != "" {
		a.Backend, err = backendFunc(a.BackendName, a.Name, a)
		return err
	}
	return nil
}

func (a *AuthZExternal) HCLBody() *hclsyntax.Body {
	return a.Remain.(*hclsyntax.Body)
}
