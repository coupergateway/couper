// Package shared provides common utilities for Couper config generators.
// It extracts schema information from Go structs using reflection and HCL tags.
package shared

import (
	"reflect"
	"strings"

	"github.com/coupergateway/couper/config/meta"
)

// AttributesMap maps meta attribute type names to their struct fields.
// Used to expand embedded meta types into their individual fields.
var AttributesMap = map[string][]reflect.StructField{
	"RequestHeadersAttributes":  NewFields(&meta.RequestHeadersAttributes{}),
	"ResponseHeadersAttributes": NewFields(&meta.ResponseHeadersAttributes{}),
	"FormParamsAttributes":      NewFields(&meta.FormParamsAttributes{}),
	"QueryParamsAttributes":     NewFields(&meta.QueryParamsAttributes{}),
	"LogFieldsAttribute":        NewFields(&meta.LogFieldsAttribute{}),
}

// NewFields extracts struct fields from the given interface
func NewFields(impl interface{}) []reflect.StructField {
	it := reflect.TypeOf(impl).Elem()
	var fields []reflect.StructField
	for i := 0; i < it.NumField(); i++ {
		fields = append(fields, it.Field(i))
	}
	return fields
}

// CollectFields recursively collects all struct fields, expanding anonymous/embedded fields
func CollectFields(t reflect.Type, fields []reflect.StructField) []reflect.StructField {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			fields = CollectFields(field.Type, fields)
		} else {
			fields = append(fields, field)
		}
	}
	return fields
}

// HCLTagInfo contains parsed information from an hcl struct tag
type HCLTagInfo struct {
	Name          string
	IsBlock       bool
	IsLabel       bool
	IsOptional    bool
	LabelOptional bool
	IsRemain      bool
}

// ParseHCLTag parses an hcl struct tag value
func ParseHCLTag(tag string) HCLTagInfo {
	info := HCLTagInfo{}
	if tag == "" {
		return info
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return info
	}

	info.Name = parts[0]

	for _, part := range parts[1:] {
		switch part {
		case "block":
			info.IsBlock = true
		case "label":
			info.IsLabel = true
		case "label_optional":
			info.IsLabel = true
			info.LabelOptional = true
		case "optional":
			info.IsOptional = true
		case "remain":
			info.IsRemain = true
		}
	}

	return info
}

// GoTypeToSchemaType converts a Go type to the corresponding schema type string
func GoTypeToSchemaType(t reflect.Type) string {
	typeStr := strings.Replace(t.String(), "*", "", 1)

	// Handle special types
	if typeStr == "config.List" {
		typeStr = "[]string"
	}

	// Handle slice types
	if strings.HasPrefix(typeStr, "[]") {
		return "tuple"
	}

	// Handle map types
	if strings.HasPrefix(typeStr, "map[") {
		return "object"
	}

	// Handle numeric types
	if strings.Contains(typeStr, "int") || strings.Contains(typeStr, "float") {
		return "number"
	}

	// Handle known types
	switch typeStr {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "hcl.Expression":
		return "expression"
	default:
		return "object"
	}
}
