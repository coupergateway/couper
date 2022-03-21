package configload

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/v2"
)

// https://datatracker.ietf.org/doc/html/rfc7231#section-4
// https://datatracker.ietf.org/doc/html/rfc7230#section-3.2.6
var methodRegExp = regexp.MustCompile("^[!#$%&'*+\\-.\\^_`|~0-9a-zA-Z]+$")

func validLabelName(name string, hr *hcl.Range) error {
	if !regexProxyRequestLabel.MatchString(name) {
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "label contains invalid character(s), allowed are 'a-z', 'A-Z', '0-9' and '_'",
			Subject:  hr,
		}}
	}
	return nil
}

func uniqueLabelName(unique map[string]struct{}, name string, hr *hcl.Range) error {
	if _, exist := unique[name]; exist {
		if name == defaultNameLabel {
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "proxy and request labels are required and only one 'default' label is allowed",
				Subject:  hr,
			}}
		}
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("proxy and request labels are required and must be unique: %q", name),
			Subject:  hr,
		}}
	}
	unique[name] = struct{}{}
	return nil
}

func verifyBodyAttributes(content *hcl.BodyContent) error {
	_, existsBody := content.Attributes["body"]
	_, existsFormBody := content.Attributes["form_body"]
	_, existsJsonBody := content.Attributes["json_body"]
	if existsBody && existsFormBody || existsBody && existsJsonBody || existsFormBody && existsJsonBody {
		rangeAttr := "body"
		if !existsBody {
			rangeAttr = "form_body"
		}
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "request can only have one of body, form_body or json_body attributes",
			Subject:  &content.Attributes[rangeAttr].Range,
		}}
	}
	return nil
}

func validMethods(methods []string, hr *hcl.Range) error {
	for _, method := range methods {
		if !methodRegExp.MatchString(method) {
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "method contains invalid character(s)",
				Subject:  hr,
			}}
		}
	}

	return nil
}
