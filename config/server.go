package config

// Server represents the HCL <server> block.
type Server struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	APIs                 APIs      `hcl:"api,block"`
	BasePath             string    `hcl:"base_path,optional"`
	Endpoints            Endpoints `hcl:"endpoint,block"`
	ErrorFile            string    `hcl:"error_file,optional"`
	Files                *Files    `hcl:"files,block"`
	Hosts                []string  `hcl:"hosts,optional"`
	Name                 string    `hcl:"name,label"`
	Spa                  *Spa      `hcl:"spa,block"`
}

// Servers represents a list of <Server> objects.
type Servers []*Server
