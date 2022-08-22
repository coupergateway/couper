package runtime

import (
	"fmt"
	"strings"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

type ACDefinitions map[string]*AccessControl

type AccessControl struct {
	Control      accesscontrol.AccessControl
	ErrorHandler []*config.ErrorHandler
}

func (m ACDefinitions) Add(name string, ac accesscontrol.AccessControl, eh []*config.ErrorHandler) error {
	n := strings.TrimSpace(name)
	if eval.IsReservedContextName(n) {
		return errors.Configuration.Message("accessControl uses reserved name as label")
	}
	if _, ok := m[n]; ok {
		return errors.Configuration.Message("accessControl already defined")
	}

	m[n] = &AccessControl{
		Control:      ac,
		ErrorHandler: eh,
	}
	return nil
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
