package collect

import (
	"reflect"

	"github.com/coupergateway/couper/config"
)

type ErrorHandlerSetter interface {
	Set(handler *config.ErrorHandler)
}

func ErrorHandlerSetters(block interface{}) []ErrorHandlerSetter {
	var errorSetter []ErrorHandlerSetter

	t := reflect.ValueOf(block)
	elem := t
	if t.Kind() == reflect.Ptr {
		elem = t.Elem()
	}

	if elem.Kind() != reflect.Struct {
		return errorSetter
	}

	for i := 0; i < elem.Type().NumField(); i++ {
		field := elem.Type().Field(i)
		if !elem.Field(i).CanInterface() {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Struct:
			if field.Anonymous && t.CanInterface() { // composition: type check fields parent
				setter, ok := t.Interface().(ErrorHandlerSetter)
				if ok {
					errorSetter = append(errorSetter, setter)
					return errorSetter
				}
			}

			errorSetter = append(errorSetter, ErrorHandlerSetters(elem.Field(i).Interface())...)
		case reflect.Slice:
			f := elem.Field(i)
			for s := 0; s < f.Len(); s++ {
				idx := f.Index(s)
				if idx.CanInterface() {
					errorSetter = append(errorSetter, ErrorHandlerSetters(idx.Interface())...)
				}
			}
		}
	}
	return errorSetter
}
