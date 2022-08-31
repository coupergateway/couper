package runtime

import (
	"fmt"
	"strings"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
)

type ACDefinitions map[string]*AccessControl

type AccessControl struct {
	Control      accesscontrol.AccessControl
	ErrorHandler []*config.ErrorHandler
}

func (m ACDefinitions) Add(name string, ac accesscontrol.AccessControl, eh []*config.ErrorHandler) {
	n := strings.TrimSpace(name)
	m[n] = &AccessControl{
		Control:      ac,
		ErrorHandler: eh,
	}
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
