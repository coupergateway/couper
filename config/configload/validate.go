package configload

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
	"github.com/hashicorp/hcl/v2"
)

func uniqueAttributeKey(body hcl.Body) error {
	if body == nil {
		return nil
	}

	content, _, diags := body.PartialContent(meta.AttributesSchema)
	if diags.HasErrors() {
		return diags
	}

	if content == nil || len(content.Attributes) == 0 {
		return nil
	}

	for _, metaAttr := range meta.AttributesSchema.Attributes {
		attr, ok := content.Attributes[metaAttr.Name]
		if !strings.HasPrefix(metaAttr.Name, "set_") && !strings.HasPrefix(metaAttr.Name, "add_") || !ok {
			continue
		}

		expr, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			fmt.Printf("%#v", attr.Expr)
		}

		unique := make(map[string]hcl.Range)

		for _, item := range expr.Items {
			keyRange := item.KeyExpr.Range()
			if keyRange.CanSliceBytes(configBytes) {
				key := keyRange.SliceBytes(configBytes)
				lwrKey := strings.ToLower(string(key))
				if previous, exist := unique[lwrKey]; exist {
					return hcl.Diagnostics{
						&hcl.Diagnostic{
							Subject:  &keyRange,
							Severity: hcl.DiagError,
							Summary: fmt.
								Sprintf("key must be unique: '%s' was previously defined at: %s",
									lwrKey,
									previous.String()),
						},
					}
				}
				unique[lwrKey] = keyRange
			}
		}
	}
	return nil
}

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
