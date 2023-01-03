package config

import (
	"reflect"
	"strings"
)

func BackendAttrFields(obj interface{}) []string {
	const filter = "_backend"
	var result []string

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tagName := attrFromField(f)
		if tagName != "" && strings.HasSuffix(tagName, filter) {
			result = append(result, tagName)
		}
	}

	return result
}

func AttrValueFromTagField(name string, obj interface{}) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := t.NumField() - 1; i > 0; i-- {
		if lookup, _ := t.Field(i).Tag.Lookup("hcl"); strings.HasPrefix(lookup, name) {
			fv := valueOf(obj).Field(i)
			if fv.IsNil() {
				return ""
			}
			return fv.String()
		}
	}

	return ""
}

func attrFromField(field reflect.StructField) string {
	return strings.Split(field.Tag.Get("hcl"), ",")[0]
}

func valueOf(obj interface{}) reflect.Value {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}
