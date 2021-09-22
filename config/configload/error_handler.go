package configload

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
)

type ErrorHandlerSetter interface {
	Set(handler *config.ErrorHandler)
}

type errorHandlerContent map[string]kindContent

type kindContent struct {
	body  hcl.Body
	kinds []string
}

func collectErrorHandlerSetter(block interface{}) []ErrorHandlerSetter {
	var errorSetter []ErrorHandlerSetter

	t := reflect.ValueOf(block)
	elem := t
	if t.Kind() == reflect.Ptr {
		elem = t.Elem()
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

			errorSetter = append(errorSetter, collectErrorHandlerSetter(elem.Field(i).Interface())...)
		case reflect.Slice:
			f := elem.Field(i)
			for s := 0; s < f.Len(); s++ {
				idx := f.Index(s)
				if idx.CanInterface() {
					errorSetter = append(errorSetter, collectErrorHandlerSetter(idx.Interface())...)
				}
			}
		}
	}
	return errorSetter
}

// newErrorHandlerContent reads given error_handler block contents and maps them by unique
// error kind declaration.
func newErrorHandlerContent(content *hcl.BodyContent) (errorHandlerContent, error) {
	configuredKinds := make(errorHandlerContent)

	if content == nil {
		return configuredKinds, fmt.Errorf("empty hcl content")
	}

	for _, block := range content.Blocks.OfType(errorHandler) {
		kinds, err := newKindsFromLabels(block.Labels)
		if err != nil {
			return nil, err
		}
		for _, k := range kinds {
			if _, exist := configuredKinds[k]; exist {
				return nil, hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("duplicate error type registration: %q", k),
					Subject:  &block.LabelRanges[0],
				}}
			}

			if k != errors.Wildcard && !errors.IsKnown(k) {
				subjRange := block.DefRange
				if len(block.LabelRanges) > 0 {
					subjRange = block.LabelRanges[0]
				}
				diag := &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("error type is unknown: %q", k),
					Subject:  &subjRange,
				}
				return nil, hcl.Diagnostics{diag}
			}

			configuredKinds[k] = kindContent{
				body:  block.Body,
				kinds: kinds,
			}
		}
	}

	return configuredKinds, nil
}

// newKindsFromLabels reads two possible kind formats and returns them per slice entry.
func newKindsFromLabels(labels []string) ([]string, error) {
	var allKinds []string
	for _, kinds := range labels {
		all := strings.Split(kinds, " ")
		for _, a := range all {
			if a == "" {
				return nil, errors.Configuration.Messagef("invalid format: %v", labels)
			}
		}
		allKinds = append(allKinds, all...)
	}
	if len(allKinds) == 0 {
		allKinds = append(allKinds, errors.Wildcard)
	}
	return allKinds, nil
}

func newErrorHandlerConfig(content kindContent, definedBackends Backends) (*config.ErrorHandler, error) {
	errHandlerConf := &config.ErrorHandler{Kinds: content.kinds}
	if d := gohcl.DecodeBody(content.body, envContext, errHandlerConf); d.HasErrors() {
		return nil, d
	}

	ep := &config.Endpoint{
		ErrorFile: errHandlerConf.ErrorFile,
		Response:  errHandlerConf.Response,
		Remain:    content.body,
	}

	if err := refineEndpoints(definedBackends, config.Endpoints{ep}, false); err != nil {
		return nil, err
	}

	errHandlerConf.Requests = ep.Requests
	errHandlerConf.Proxies = ep.Proxies

	return errHandlerConf, nil
}
