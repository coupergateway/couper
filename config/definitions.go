package config

type Definitions struct {
	JWT []*JWT `hcl:"jwt,block"`
}
