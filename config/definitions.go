package config

type Definitions struct {
	Jwt []*Jwt `hcl:"jwt,block"`
}
