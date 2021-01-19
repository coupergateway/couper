package config

type Definitions struct {
	BasicAuth []*BasicAuth `hcl:"basic_auth,block"`
	JWT       []*JWT       `hcl:"jwt,block"`
}
