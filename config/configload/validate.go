package configload

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
)

var regexLabel = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// https://datatracker.ietf.org/doc/html/rfc7231#section-4
// https://datatracker.ietf.org/doc/html/rfc7230#section-3.2.6
var methodRegExp = regexp.MustCompile("^[!#$%&'*+\\-.\\^_`|~0-9a-zA-Z]+$")

func newDiagErr(subject *hcl.Range, summary string) error {
	return hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Subject:  subject,
	}}
}

func validLabel(name string, subject *hcl.Range) error {
	if name == "" {
		return newDiagErr(subject, "label is empty")
	}

	if !regexLabel.MatchString(name) {
		return newDiagErr(subject, "label contains invalid character(s), allowed are 'a-z', 'A-Z', '0-9' and '_'")
	}

	return nil
}

func uniqueLabelName(scopeOfUniqueness string, unique map[string]struct{}, name string, hr *hcl.Range) error {
	if _, exist := unique[name]; exist {
		return newDiagErr(hr, fmt.Sprintf("%s names (either default or explicitly set via label) must be unique: %q", scopeOfUniqueness, name))
	}

	unique[name] = struct{}{}

	return nil
}

func verifyBodyAttributes(blockName string, content *hcl.BodyContent) error {
	_, existsBody := content.Attributes["body"]
	_, existsFormBody := content.Attributes["form_body"]
	_, existsJsonBody := content.Attributes["json_body"]

	if existsBody && existsFormBody || existsBody && existsJsonBody || existsFormBody && existsJsonBody {
		rangeAttr := "body"
		if !existsBody {
			rangeAttr = "form_body"
		}

		return newDiagErr(&content.Attributes[rangeAttr].Range,
			blockName+" can only have one of body, form_body or json_body attributes")
	}

	return nil
}

func verifyResponseBodyAttrs(b hcl.Body) error {
	content, _, _ := b.PartialContent(config.ResponseInlineSchema)
	_, existsBody := content.Attributes["body"]
	_, existsJsonBody := content.Attributes["json_body"]
	if existsBody && existsJsonBody {
		return newDiagErr(&content.Attributes["body"].Range, "response can only have one of body or json_body attributes")
	}
	return nil
}

var invalidAttributes = []string{"disable_certificate_validation", "disable_connection_reuse", "http2", "max_connections"}
var openApiBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "openapi",
		},
	},
}

func invalidRefinement(body hcl.Body) error {
	const message = "backend reference: refinement for %q is not permitted"
	attrs, _ := body.JustAttributes()
	if attrs == nil {
		return nil
	}
	for _, name := range invalidAttributes {
		attr, exist := attrs[name]
		if exist {
			return newDiagErr(&attr.NameRange, fmt.Sprintf(message, attr.Name))
		}
	}

	content, _, _ := body.PartialContent(openApiBlockSchema)
	if content != nil && len(content.Blocks.OfType("openapi")) > 0 {
		return newDiagErr(&content.Blocks.OfType("openapi")[0].DefRange, fmt.Sprintf(message, "openapi"))
	}

	return nil
}

func invalidOriginRefinement(reference, params hcl.Body) error {
	const origin = "origin"
	refAttrs, _ := reference.JustAttributes()
	paramAttrs, _ := params.JustAttributes()

	refOrigin, _ := refAttrs[origin]
	paramOrigin, _ := paramAttrs[origin]

	if paramOrigin != nil && refOrigin != nil {
		if paramOrigin.Expr != refOrigin.Expr {
			return newDiagErr(&paramOrigin.Range, "backend reference: origin must be equal")
		}
	}
	return nil
}

func validMethods(methods []string, hr *hcl.Range) error {
	for _, method := range methods {
		if !methodRegExp.MatchString(method) {
			return newDiagErr(hr, "method contains invalid character(s)")
		}
	}

	return nil
}
