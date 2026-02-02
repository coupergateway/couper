package shared

import (
	"reflect"

	"github.com/coupergateway/couper/config"
)

// GetInlineFields returns fields from both the main struct and its Inline() type
func GetInlineFields(impl interface{}) []reflect.StructField {
	t := reflect.TypeOf(impl).Elem()

	var fields []reflect.StructField
	fields = CollectFields(t, fields)

	// Check if the type implements the Inline interface
	if inlineType, ok := impl.(config.Inline); ok {
		it := reflect.TypeOf(inlineType.Inline()).Elem()
		for i := 0; i < it.NumField(); i++ {
			field := it.Field(i)
			// Check if this field is a meta attribute type that should be expanded
			if metaFields, ok := AttributesMap[field.Name]; ok {
				fields = append(fields, metaFields...)
			} else {
				fields = append(fields, field)
			}
		}
	}

	return fields
}

// IsInlineType checks if the given interface implements the Inline interface
func IsInlineType(impl interface{}) bool {
	_, ok := impl.(config.Inline)
	return ok
}
