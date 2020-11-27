package config

type Definitions struct {
	Backend   []*Backend   `hcl:"backend,block"`
	BasicAuth []*BasicAuth `hcl:"basic_auth,block"`
	JWT       []*JWT       `hcl:"jwt,block"`
}
