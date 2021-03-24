package runtime

import (
	"fmt"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
)

type ACDefinitions map[string]*AccessControl

type AccessControl struct {
	ValidateFn   accesscontrol.AccessControl
	ErrorHandler []*config.ErrorHandler
}

func (m ACDefinitions) MustExist(name string) error {
	if m == nil {
		panic("no accessControl configuration")
	}

	if _, ok := m[name]; !ok {
		return fmt.Errorf("accessControl is not defined: " + name)
	}

	return nil
}
