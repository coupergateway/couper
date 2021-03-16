package config

// Server represents the <Server> object.
type Server struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	APIs                 APIs      `hcl:"api,block"`
	BasePath             string    `hcl:"base_path,optional"`
	CORS                 *CORS     `hcl:"cors,block"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Files                *Files    `hcl:"files,block"`
	Hosts                []string  `hcl:"hosts,optional"`
	Name                 string    `hcl:"name,label"`
	Spa                  *Spa      `hcl:"spa,block"`
}

// Servers represents a list of <Server> objects.
type Servers []*Server
