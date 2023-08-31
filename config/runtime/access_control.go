package runtime

import (
	"strings"

	"github.com/coupergateway/couper/accesscontrol"
	"github.com/coupergateway/couper/config"
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
