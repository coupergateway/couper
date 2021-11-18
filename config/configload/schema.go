package configload

import (
	"reflect"
	"regexp"

	"github.com/avenga/couper/config"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

const (
	noLabelForErrorHandler = "No labels are expected for error_handler blocks."
	summUnsupportedAttr    = "Unsupported argument"
	summUnsupportedBlock   = "Unsupported block type"
)

var (
	reFetchUnsupportedName = regexp.MustCompile(`\"([^"]+)\"`)
	reFetchLabeledName     = regexp.MustCompile(`All (.*) blocks must have .* labels \(.*\).`)
	reFetchUnlabeledName   = regexp.MustCompile(`No labels are expected for (.*) blocks.`)
	reFetchUnexpectedArg   = regexp.MustCompile(`An argument named (.*) is not expected here.`)
)

func ValidateConfigSchema(body hcl.Body, obj interface{}) hcl.Diagnostics {
	attrs, blocks, diags := getSchemaComponents(body, obj)
	diags = filterValidErrors(attrs, blocks, diags)

	for _, block := range blocks {
		diags = diags.Extend(checkObjectFields(block, obj))
	}

	return uniqueErrors(diags)
}

func filterValidErrors(attrs hcl.Attributes, blocks hcl.Blocks, diags hcl.Diagnostics) hcl.Diagnostics {
	var errors hcl.Diagnostics

	for _, err := range diags {
		if err.Detail == noLabelForErrorHandler {
			continue
		}

		matches := reFetchUnsupportedName.FindStringSubmatch(err.Detail)
		if len(matches) != 2 {
			if match := reFetchLabeledName.MatchString(err.Detail); match {
				errors = errors.Append(err)
				continue
			}
			if match := reFetchUnlabeledName.MatchString(err.Detail); match {
				errors = errors.Append(err)
				continue
			}
			if match := reFetchUnexpectedArg.MatchString(err.Detail); match {
				errors = errors.Append(err)
				continue
			}

			errors = errors.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Subject:  err.Subject,
				Summary:  "cannot match argument name from: " + err.Detail,
			})

			continue
		}

		name := matches[1]

		if err.Summary == summUnsupportedAttr {
			if _, ok := attrs[name]; ok {
				continue
			}
		} else if err.Summary == summUnsupportedBlock {
			if len(blocks.OfType(name)) > 0 {
				continue
			}
		}

		errors = errors.Append(err)
	}

	return errors
}

func checkObjectFields(block *hcl.Block, obj interface{}) hcl.Diagnostics {
	var errors hcl.Diagnostics
	var checked bool

	typ := reflect.TypeOf(obj)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if field.Anonymous {
			o := reflect.New(field.Type).Interface()
			errors = errors.Extend(checkObjectFields(block, o))

			continue
		}

		// TODO: How to implement this automatically?
		if field.Type.String() != "*config.OAuth2ReqAuth" || block.Type != "oauth2" || typ.String() == "config.Backend" {
			if _, ok := field.Tag.Lookup("hcl"); !ok {
				continue
			}
			if field.Tag.Get("hcl") != block.Type+",block" {
				continue
			}
		}

		checked = true

		if field.Type.Kind() == reflect.Ptr {
			o := reflect.New(field.Type.Elem()).Interface()
			errors = errors.Extend(ValidateConfigSchema(block.Body, o))

			continue
		} else if field.Type.Kind() == reflect.Slice {
			tp := reflect.TypeOf(val.Field(i).Interface())
			if tp.Kind() == reflect.Slice {
				tp = tp.Elem()
			}

			vl := reflect.ValueOf(tp)
			if vl.Kind() == reflect.Ptr {
				vl = vl.Elem()
			}

			if vl.Kind() == reflect.Struct {
				var elem reflect.Type

				if tp.Kind() == reflect.Struct {
					elem = tp
				} else if tp.Kind() == reflect.Ptr {
					elem = tp.Elem()
				} else {
					errors = errors.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Unsupported type.Kind '" + tp.Kind().String() + "' for: " + field.Name,
					})

					continue
				}

				o := reflect.New(elem).Interface()
				errors = errors.Extend(ValidateConfigSchema(block.Body, o))

				continue
			}
		}

		errors = errors.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "A block without config test found: " + field.Name,
		})
	}

	if !checked {
		if i, ok := obj.(config.Inline); ok {
			errors = errors.Extend(checkObjectFields(block, i.Inline()))
		}
	}

	return errors
}

func getSchemaComponents(body hcl.Body, obj interface{}) (hcl.Attributes, hcl.Blocks, hcl.Diagnostics) {
	var (
		attrs  = make(hcl.Attributes)
		blocks hcl.Blocks
		errors hcl.Diagnostics
	)

	schema, _ := gohcl.ImpliedBodySchema(obj)

	typ := reflect.TypeOf(obj)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// TODO: How to implement this automatically?
	if typ.String() == "config.Backend" {
		schema = config.SchemaWithOAuth2RA(schema)
	}

	if _, ok := obj.(ErrorHandlerSetter); ok {
		schema = config.WithErrorHandlerSchema(schema)
	}

	attrs, blocks, errors = completeSchemaComponents(body, schema, attrs, blocks, errors)

	if i, ok := obj.(config.Inline); ok {
		attrs, blocks, errors = completeSchemaComponents(body, i.Schema(true), attrs, blocks, errors)
	}

	return attrs, blocks, errors
}

func completeSchemaComponents(body hcl.Body, schema *hcl.BodySchema, attrs hcl.Attributes,
	blocks hcl.Blocks, errors hcl.Diagnostics) (hcl.Attributes, hcl.Blocks, hcl.Diagnostics) {

	content, diags := body.Content(schema)

	for _, diag := range diags {
		// TODO: How to implement this block automatically?
		if match := reFetchLabeledName.MatchString(diag.Detail); match {
			bodyContent := bodyToContent(body)

			added := false
			for _, block := range bodyContent.Blocks {
				switch block.Type {
				case "api", "backend", "proxy", "request", "server":
					blocks = append(blocks, block)

					added = true
				}
			}

			if !added {
				errors = errors.Append(diag)
			}
		} else {
			errors = errors.Append(diag)
		}
	}

	if content != nil {
		for name, attr := range content.Attributes {
			attrs[name] = attr
		}

		blocks = append(blocks, content.Blocks...)
	}

	return attrs, blocks, errors
}

func uniqueErrors(errors hcl.Diagnostics) hcl.Diagnostics {
	var unique hcl.Diagnostics

	for _, diag := range errors {
		var contains bool

		for _, is := range unique {
			if reflect.DeepEqual(diag, is) {
				contains = true
				break
			}
		}

		if !contains {
			unique = unique.Append(diag)
		}
	}

	return unique
}
