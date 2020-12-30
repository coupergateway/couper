package config

import "fmt"

type Definitions struct {
	Backend   []*Backend   `hcl:"backend,block"`
	BasicAuth []*BasicAuth `hcl:"basic_auth,block"`
	JWT       []*JWT       `hcl:"jwt,block"`
}

func (d *Definitions) BackendWithName(name string) (*Backend, error) {
	if name == "" {
		return nil, nil
	}

	for _, backend := range d.Backend {
		if backend.Name == name {
			return backend, nil
		}
	}
	return nil, fmt.Errorf("missing backend reference: %q", name)
}
