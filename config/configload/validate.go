package configload

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/utils"
)

var (
	regexLabel     = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	reCleanPattern = regexp.MustCompile(`{([^}]+)}`)
)

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

func validateBody(body hcl.Body, afterMerge bool) error {
	hsBody, ok := body.(*hclsyntax.Body)
	if !ok {
		return fmt.Errorf("body must be hclsyntax.Body")
	}

	for _, outerBlock := range hsBody.Blocks {
		if outerBlock.Type == definitions {
			uniqueBackends := make(map[string]struct{})
			uniqueACs := make(map[string]struct{})
			uniqueJWTSigningProfiles := make(map[string]struct{})
			uniqueProxies := make(map[string]struct{})
			for _, innerBlock := range outerBlock.Body.Blocks {
				if !afterMerge {
					if len(innerBlock.Labels) == 0 {
						return newDiagErr(&innerBlock.OpenBraceRange, "missing label")
					}
				}
				label := innerBlock.Labels[0]
				label = strings.TrimSpace(label)
				labelRange := innerBlock.LabelRanges[0]

				switch innerBlock.Type {
				case backend:
					if !afterMerge {
						if err := validLabel(label, &labelRange); err != nil {
							return err
						}

						if strings.HasPrefix(label, "anonymous_") {
							return newDiagErr(&labelRange, "backend label must not start with 'anonymous_'")
						}

						if _, set := uniqueBackends[label]; set {
							return newDiagErr(&labelRange, "backend labels must be unique")
						}
						uniqueBackends[label] = struct{}{}
					}
				case "basic_auth", "beta_oauth2", "oidc", "saml":
					err := checkAC(uniqueACs, label, labelRange, afterMerge)
					if err != nil {
						return err
					}
				case "jwt":
					err := checkAC(uniqueACs, label, labelRange, afterMerge)
					if err != nil {
						return err
					}

					attrs, _ := innerBlock.Body.JustAttributes() // just get attributes, ignore diags for now
					if _, set := attrs["signing_ttl"]; set {
						err = checkJWTSigningProfiles(uniqueJWTSigningProfiles, label, labelRange)
						if err != nil {
							return err
						}
					}
				case "jwt_signing_profile":
					err := checkJWTSigningProfiles(uniqueJWTSigningProfiles, label, labelRange)
					if err != nil {
						return err
					}
				case proxy:
					if _, set := uniqueProxies[label]; set {
						return newDiagErr(&labelRange, "proxy labels must be unique")
					}
					uniqueProxies[label] = struct{}{}
				}
			}
		} else if outerBlock.Type == server {
			uniqueEndpoints := make(map[string]struct{})
			serverBasePath, err := getBasePath(outerBlock)
			if err != nil {
				return err
			}

			serverBasePath = path.Join("/", serverBasePath)
			for _, innerBlock := range outerBlock.Body.Blocks {
				if innerBlock.Type == endpoint {
					if err = registerEndpointPattern(uniqueEndpoints, serverBasePath, innerBlock); err != nil {
						return err
					}
				} else if innerBlock.Type == api {
					apiBasePath, err := getBasePath(innerBlock)
					if err != nil {
						return err
					}

					apiBasePath = path.Join(serverBasePath, apiBasePath)
					for _, innerInnerBlock := range innerBlock.Body.Blocks {
						if innerInnerBlock.Type == endpoint {
							if err = registerEndpointPattern(uniqueEndpoints, apiBasePath, innerInnerBlock); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func getBasePath(bl *hclsyntax.Block) (string, error) {
	basePath := ""
	if bp, set := bl.Body.Attributes["base_path"]; set {
		bpv, diags := bp.Expr.Value(nil)
		if diags.HasErrors() {
			return "", diags
		}
		if bpv.Type() != cty.String {
			sr := bp.Expr.StartRange()
			return "", newDiagErr(&sr, "base_path must evaluate to string")
		}
		basePath = bpv.AsString()
	}

	return basePath, nil
}

func registerEndpointPattern(endpointPatterns map[string]struct{}, basePath string, bl *hclsyntax.Block) error {
	pattern := utils.JoinOpenAPIPath(basePath, bl.Labels[0])
	pattern = reCleanPattern.ReplaceAllString(pattern, "{}")
	if _, set := endpointPatterns[pattern]; set {
		return newDiagErr(&bl.LabelRanges[0], "duplicate endpoint")
	}

	endpointPatterns[pattern] = struct{}{}
	return nil
}

func checkAC(acLabels map[string]struct{}, label string, labelRange hcl.Range, afterMerge bool) error {
	if !afterMerge {
		if label == "" {
			return newDiagErr(&labelRange, "accessControl requires a label")
		}

		if eval.IsReservedContextName(label) {
			return newDiagErr(&labelRange, "accessControl uses reserved name as label")
		}
	}

	if _, set := acLabels[label]; set {
		return newDiagErr(&labelRange, "AC labels must be unique")
	}

	acLabels[label] = struct{}{}
	return nil
}

func checkJWTSigningProfiles(spLabels map[string]struct{}, label string, labelRange hcl.Range) error {
	if _, set := spLabels[label]; set {
		return newDiagErr(&labelRange, "JWT signing profile labels must be unique")
	}

	spLabels[label] = struct{}{}
	return nil
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

func uniqueLabelName(unique map[string]struct{}, name string, hr *hcl.Range) error {
	if _, exist := unique[name]; exist {
		if name == defaultNameLabel {
			return newDiagErr(hr, "proxy and request labels are required and only one 'default' label is allowed")
		}

		return newDiagErr(hr, fmt.Sprintf("proxy and request labels are required and must be unique: %q", name))
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
