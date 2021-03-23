package config

import "github.com/hashicorp/hcl/v2"

type AccessControlSetter struct {
	ErrorHandler []*ErrorHandler
}

func (acs *AccessControlSetter) Set(kinds []string, body hcl.Body) {
	acs.ErrorHandler = append(acs.ErrorHandler, &ErrorHandler{
		Kinds:  kinds,
		Remain: body,
	})
}
