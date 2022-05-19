package seetie

import "github.com/zclconf/go-cty/cty"

type Object interface {
	Value() cty.Value
}
