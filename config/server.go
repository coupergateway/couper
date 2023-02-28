package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/config/schema"
)

var _ schema.BodySchema = &Server{}

// Server represents the <Server> object.
type Server struct {
	AccessControl        []string    `hcl:"access_control,optional" docs:"The [access controls](../access-control) to protect the server. Inherited by nested blocks."`
	APIs                 APIs        `hcl:"api,block" docs:"Configures an API (zero or more)."`
	BasePath             string      `hcl:"base_path,optional" docs:"The path prefix for all requests."`
	CORS                 *CORS       `hcl:"cors,block" docs:"Configures [CORS](/configuration/block/cors) settings (zero or one)."`
	DisableAccessControl []string    `hcl:"disable_access_control,optional" docs:"Disables access controls by name."`
	Endpoints            Endpoints   `hcl:"endpoint,block" docs:"Configures a free [endpoint](/configuration/block/endpoint) (zero or more)."`
	ErrorFile            string      `hcl:"error_file,optional" docs:"Location of the error file template."`
	Files                FilesBlocks `hcl:"files,block" docs:"Configures file serving (zero or more)."`
	Hosts                []string    `hcl:"hosts,optional" docs:"Mandatory, if there is more than one {server} block."`
	Name                 string      `hcl:"name,label,optional"`
	Remain               hcl.Body    `hcl:",remain"`
	SPAs                 SPAs        `hcl:"spa,block" docs:"Configures an SPA (zero or more)."`
	TLS                  *ServerTLS  `hcl:"tls,block" docs:"Configures [server TLS](/configuration/block/server_tls) (zero or one)."`
}

// Servers represents a list of <Server> objects.
type Servers []*Server

func (s Server) Schema() *hcl.BodySchema {
	bs, _ := gohcl.ImpliedBodySchema(s)
	return meta.MergeSchemas(bs, meta.ResponseHeadersAttributesSchema, meta.LogFieldsAttributeSchema)
}
