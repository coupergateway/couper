package config

type Plugin struct {
	File string `hcl:"file"`
	Name string `hcl:"name,label"`
}
