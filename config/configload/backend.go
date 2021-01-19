package configload

import "github.com/hashicorp/hcl/v2"

type Backends []*Backend

type Backend struct {
	Name string
	Config hcl.Body
}

func (b Backends) WithName(name string) hcl.Body {
	for _, item := range b {
		if item.Name == name {
			return item.Config
		}
	}
	return nil
}
