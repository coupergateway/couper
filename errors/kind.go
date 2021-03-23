package errors

import "github.com/avenga/couper/errors/accesscontrol/jwt"

func KindToType(string) error {
	var err error
	// TODO: generate and combine /w stringer
	err = jwt.TokenExpired

	if err == nil {
		panic("unknown error type")
	}
	return err
}
