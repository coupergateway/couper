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
					if !afterMerge {
						if _, set := uniqueProxies[label]; set {
							return newDiagErr(&labelRange, "proxy labels must be unique")
						}
						uniqueProxies[label] = struct{}{}
					}
				}
			}
		} else if outerBlock.Type == server {
			uniqueEndpoints := make(map[string]struct{})
			serverBasePath, err := getBasePath(outerBlock, afterMerge)
			if err != nil {
				return err
			}

			serverBasePath = path.Join("/", serverBasePath)
			for _, innerBlock := range outerBlock.Body.Blocks {
				if innerBlock.Type == endpoint {
					if err = registerEndpointPattern(uniqueEndpoints, serverBasePath, innerBlock, afterMerge); err != nil {
						return err
					}
				} else if innerBlock.Type == api {
					apiBasePath, err := getBasePath(innerBlock, afterMerge)
					if err != nil {
						return err
					}

					apiBasePath = path.Join(serverBasePath, apiBasePath)
					for _, innerInnerBlock := range innerBlock.Body.Blocks {
						if innerInnerBlock.Type == endpoint {
							if err = registerEndpointPattern(uniqueEndpoints, apiBasePath, innerInnerBlock, afterMerge); err != nil {
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

func checkPathSegments(pathType, path string, r hcl.Range) error {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return newDiagErr(&r, pathType+` must not contain "." or ".." segments`)
		}
	}
	return nil
}

func getBasePath(bl *hclsyntax.Block, afterMerge bool) (string, error) {
	basePath := ""
	if bp, set := bl.Body.Attributes["base_path"]; set {
		bpv, diags := bp.Expr.Value(nil)
		if diags.HasErrors() {
			return "", diags
		}
		r := bp.Expr.StartRange()
		if !afterMerge && bpv.Type() != cty.String {
			return "", newDiagErr(&r, "base_path must evaluate to string")
		}
		basePath = bpv.AsString()
		if !afterMerge {
			if err := checkPathSegments("base_path", basePath, r); err != nil {
				return "", err
			}
		}
	}

	return basePath, nil
}

func registerEndpointPattern(endpointPatterns map[string]struct{}, basePath string, bl *hclsyntax.Block, afterMerge bool) error {
	pattern := bl.Labels[0]
	if !afterMerge {
		if !strings.HasPrefix(pattern, "/") {
			return newDiagErr(&bl.LabelRanges[0], `endpoint path pattern must start with "/"`)
		}

		if err := checkPathSegments("endpoint path pattern", pattern, bl.LabelRanges[0]); err != nil {
			return err
		}
	}

	pattern = utils.JoinOpenAPIPath(basePath, pattern)
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

func uniqueLabelName(scopeOfUniqueness string, unique map[string]struct{}, name string, hr *hcl.Range) error {
	if _, exist := unique[name]; exist {
		return newDiagErr(hr, fmt.Sprintf("%s names (either default or explicitly set via label) must be unique: %q", scopeOfUniqueness, name))
	}

	unique[name] = struct{}{}

	return nil
}

func verifyBodyAttributes(blockName string, body *hclsyntax.Body) error {
	_, existsBody := body.Attributes["body"]
	_, existsFormBody := body.Attributes["form_body"]
	_, existsJSONBody := body.Attributes["json_body"]

	if existsBody && existsFormBody || existsBody && existsJSONBody || existsFormBody && existsJSONBody {
		rangeAttr := "body"
		if !existsBody {
			rangeAttr = "form_body"
		}

		r := body.Attributes[rangeAttr].Range()
		return newDiagErr(&r,
			blockName+" can only have one of body, form_body or json_body attributes")
	}

	return nil
}

func verifyResponseBodyAttrs(b hcl.Body) error {
	content, _, _ := b.PartialContent(config.ResponseInlineSchema)
	_, existsBody := content.Attributes["body"]
	_, existsJSONBody := content.Attributes["json_body"]
	if existsBody && existsJSONBody {
		return newDiagErr(&content.Attributes["body"].Range, "response can only have one of body or json_body attributes")
	}
	return nil
}

var invalidAttributes = []string{"disable_certificate_validation", "disable_connection_reuse", "http2", "max_connections"}
var forbiddenInRefinedBackendBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "openapi",
		},
		config.OAuthBlockHeaderSchema,
		{
			Type: "beta_health",
		},
		config.TokenRequestBlockHeaderSchema,
		{
			Type: "beta_rate_limit",
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

	content, _, _ := body.PartialContent(forbiddenInRefinedBackendBlockSchema)
	if content != nil && len(content.Blocks) > 0 {
		return newDiagErr(&content.Blocks[0].DefRange, fmt.Sprintf(message, content.Blocks[0].Type))
	}

	return nil
}

func invalidOriginRefinement(reference, params hcl.Body) error {
	const origin = "origin"
	refAttrs, _ := reference.JustAttributes()
	paramAttrs, _ := params.JustAttributes()

	refOrigin := refAttrs[origin]
	paramOrigin := paramAttrs[origin]

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
