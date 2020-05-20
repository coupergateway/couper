package config

type Frontend struct {
	Endpoint []*Endpoint `hcl:"endpoint,block"`
	Name     string      `hcl:"name,label"`
	BasePath string      `hcl:"base_path,attr"`
}
