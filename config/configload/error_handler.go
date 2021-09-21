package configload

import "github.com/avenga/couper/config"

type ErrorHandlerSetter interface {
	Set(handler *config.ErrorHandler)
}
