package configload

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload/collect"
	"github.com/avenga/couper/errors"
)

type kindContent struct {
	body  hcl.Body
	kinds []string
}

func configureErrorHandler(setter []collect.ErrorHandlerSetter, definedBackends Backends) error {
	for _, ehs := range setter {
		body, ok := ehs.(config.Body)
		if !ok {
			continue
		}

		kinds, ehc, err := newErrorHandlerContent(bodyToContent(body.HCLBody()))
		if err != nil {
			return err
		}

		for _, hc := range ehc {
			errHandlerConf, confErr := newErrorHandlerConfig(hc, definedBackends)
			if confErr != nil {
				return confErr
			}

			ehs.Set(errHandlerConf)
		}

		if handler, has := ehs.(config.ErrorHandlerGetter); has {
			defaultHandler := handler.DefaultErrorHandler()
			_, exist := kinds[errors.Wildcard]
			if !exist {
				for _, kind := range defaultHandler.Kinds {
					_, exist = kinds[kind]
					if exist {
						break
					}
				}
			}

			if !exist {
				ehs.Set(handler.DefaultErrorHandler())
			}
		}
	}
	return nil
}

// newErrorHandlerContent reads given error_handler block contents and maps them by unique
// error kind declaration.
func newErrorHandlerContent(content *hcl.BodyContent) (map[string]struct{}, []kindContent, error) {
	if content == nil {
		return nil, nil, fmt.Errorf("empty hcl content")
	}

	configuredKinds := make(map[string]struct{})
	var kindContents []kindContent

	for _, block := range content.Blocks.OfType(errorHandler) {
		kinds, err := newKindsFromLabels(block)
		if err != nil {
			return nil, nil, err
		}
		for _, k := range kinds {
			if _, exist := configuredKinds[k]; exist {
				return nil, nil, hcl.Diagnostics{&hcl.Diagnostic{
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
				return nil, nil, hcl.Diagnostics{diag}
			}

			configuredKinds[k] = struct{}{}
		}
		kindContents = append(kindContents, kindContent{
			body:  block.Body,
			kinds: kinds,
		})
	}

	return configuredKinds, kindContents, nil
}

const errorHandlerLabelSep = " "

// newKindsFromLabels reads two possible kind formats and returns them per slice entry.
func newKindsFromLabels(block *hcl.Block) ([]string, error) {
	var allKinds []string
	for _, kinds := range block.Labels {
		all := strings.Split(kinds, errorHandlerLabelSep)
		for i, a := range all {
			if a == "" {
				err := hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "empty error_handler label",
					Subject:  &block.LabelRanges[i],
				}
				return nil, errors.Configuration.Message(err.Error())
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
