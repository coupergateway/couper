package configload

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/configload/collect"
	"github.com/coupergateway/couper/config/meta"
	"github.com/coupergateway/couper/internal/seetie"
)

const (
	noLabelForErrorHandler = "No labels are expected for error_handler blocks."
	summUnsupportedAttr    = "Unsupported argument"
)

var reFetchUnexpectedArg = regexp.MustCompile(`An argument named (.*) is not expected here\.`)

func validateConfigSchema(body hcl.Body, obj interface{}) hcl.Diagnostics {
	blocks, diags := getSchemaComponents(body, obj)
	diags = enhanceErrors(diags, obj)

	for _, block := range blocks {
		diags = diags.Extend(checkObjectFields(block, obj))
	}

	return uniqueErrors(diags)
}

// enhanceErrors enhances diagnostics e.g. by providing a hint how to solve the issue
func enhanceErrors(diags hcl.Diagnostics, obj interface{}) hcl.Diagnostics {
	_, isEndpoint := obj.(*config.Endpoint)
	_, isProxy := obj.(*config.Proxy)
	for _, err := range diags {
		if err.Summary == summUnsupportedAttr && (isEndpoint || isProxy) {
			if matches := reFetchUnexpectedArg.FindStringSubmatch(err.Detail); matches != nil && matches[1] == `"path"` {
				err.Detail = err.Detail + ` Use the "path" attribute in a backend block instead.`
			}
		}
	}
	return diags
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
			errors = errors.Extend(validateConfigSchema(block.Body, o))

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
				errors = errors.Extend(validateConfigSchema(block.Body, o))

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

func getSchemaComponents(body hcl.Body, obj interface{}) (hcl.Blocks, hcl.Diagnostics) {
	var (
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
		meta.MergeSchemas(schema, config.OAuthBlockSchema, config.TokenRequestBlockSchema)
	}

	if _, ok := obj.(collect.ErrorHandlerSetter); ok {
		schema = config.WithErrorHandlerSchema(schema)
	}

	if i, ok := obj.(config.Inline); ok {
		inlineSchema := i.Schema(true)
		schema.Attributes = append(schema.Attributes, inlineSchema.Attributes...)
		schema.Blocks = append(schema.Blocks, inlineSchema.Blocks...)
	}

	blocks, errors = completeSchemaComponents(body, schema, blocks, errors)

	return blocks, errors
}

func completeSchemaComponents(body hcl.Body, schema *hcl.BodySchema,
	blocks hcl.Blocks, errors hcl.Diagnostics) (hcl.Blocks, hcl.Diagnostics) {

	content, diags := body.Content(schema)

	errorHandlerCompleted := false

	for _, diag := range diags {
		// TODO: How to implement this block automatically?
		if diag.Detail == noLabelForErrorHandler {
			if errorHandlerCompleted {
				continue
			}

			bodyContent := bodyToContent(body.(*hclsyntax.Body))

			for _, block := range bodyContent.Blocks {
				if block.Type == errorHandler && len(block.Labels) > 0 {
					blocks = append(blocks, block)
				}
			}

			errorHandlerCompleted = true
		} else {
			errors = errors.Append(diag)
		}
	}

	if content != nil {
		for name, attr := range content.Attributes {
			if expr, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {

				value, _ := attr.Expr.Value(nil)
				if value.CanIterateElements() {
					unique := make(map[string]struct{})

					iter := value.ElementIterator()

					for {
						if !iter.Next() {
							break
						}

						k, _ := iter.Element()
						if k.Type() != cty.String {
							continue
						}

						keyName := seetie.ValueToString(k)
						switch name {
						case "add_request_headers", "add_response_headers", "required_permission", "headers", "set_request_headers", "set_response_headers":
							// header field names, method names: handle object keys case-insensitively
							keyName = strings.ToLower(keyName)
						}
						if _, ok := unique[keyName]; ok {
							errors = errors.Append(&hcl.Diagnostic{
								Subject:  &expr.SrcRange,
								Severity: hcl.DiagError,
								Summary:  fmt.Sprintf("key in an attribute must be unique: '%s'", keyName),
								Detail:   "Key must be unique for " + keyName + ".",
							})
						}

						unique[keyName] = struct{}{}
					}
				}
			}
		}

		blocks = append(blocks, content.Blocks...)
	}

	return blocks, errors
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

func bodyToContent(b *hclsyntax.Body) *hcl.BodyContent {
	content := &hcl.BodyContent{
		MissingItemRange: *getRange(b),
	}

	if len(b.Attributes) > 0 {
		content.Attributes = make(hcl.Attributes)
	}
	for name, attr := range b.Attributes {
		content.Attributes[name] = &hcl.Attribute{
			Name:      attr.Name,
			Expr:      attr.Expr,
			Range:     attr.Range(),
			NameRange: attr.NameRange,
		}
	}

	for _, block := range b.Blocks {
		content.Blocks = append(content.Blocks, &hcl.Block{
			Body:        block.Body,
			DefRange:    block.DefRange(),
			LabelRanges: block.LabelRanges,
			Labels:      block.Labels,
			Type:        block.Type,
			TypeRange:   block.TypeRange,
		})
	}

	return content
}

func getRange(body *hclsyntax.Body) *hcl.Range {
	if body == nil {
		return &hcl.Range{}
	}

	return &body.SrcRange
}
